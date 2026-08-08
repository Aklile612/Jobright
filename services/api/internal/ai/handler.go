package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jobright/api/internal/atsscore"
	"github.com/jobright/api/internal/auth"
	"github.com/jobright/api/internal/gemini"
	"github.com/jobright/api/internal/groq"
	"github.com/jobright/api/internal/middleware"
	"github.com/jobright/api/internal/models"
	"github.com/jobright/api/internal/pdfstamp"
	"github.com/jobright/api/pkg/response"
	"gorm.io/gorm"
)

type Handler struct {
	db         *gorm.DB
	auth       *auth.Service
	gemini     *gemini.Client
	groq       *groq.Client
	uploadDir  string
	pythonBin  string
	stampScript string
}

func NewHandler(db *gorm.DB, authSvc *auth.Service, geminiClient *gemini.Client, groqClient *groq.Client, uploadDir string) *Handler {
	h := &Handler{
		db:        db,
		auth:      authSvc,
		gemini:    geminiClient,
		groq:      groqClient,
		uploadDir: uploadDir,
	}
	h.pythonBin, h.stampScript = resolveStampTools()
	return h
}

type jobContextRequest struct {
	JobID           string   `json:"job_id"`
	Title           string   `json:"title"`
	Company         string   `json:"company"`
	Description     string   `json:"description"`
	Tone            string   `json:"tone"`
	Extra           string   `json:"extra"`
	MissingKeywords []string `json:"missing_keywords"`
	MissingSkills   []string `json:"missing_skills"`
	Suggestions     []string `json:"suggestions"`
}

type analyzeResult struct {
	MatchScore      float64  `json:"match_score"`
	MissingKeywords []string `json:"missing_keywords"`
	MissingSkills   []string `json:"missing_skills"`
	Strengths       []string `json:"strengths"`
	Suggestions     []string `json:"suggestions"`
	Summary         string   `json:"summary"`
	Covered         int      `json:"covered,omitempty"`
	TotalKeywords   int      `json:"total_keywords,omitempty"`
	Model           string   `json:"model"`
}

type tailorResult struct {
	Headline         string        `json:"headline"`
	Summary          string        `json:"summary"`
	Skills           []string      `json:"skills"`
	ResumeMarkdown   string        `json:"resume_markdown"`
	CoverLetter      string        `json:"cover_letter"`
	Analyze          analyzeResult `json:"analyze"`
	Model            string        `json:"model"`
	OriginalResumeID string        `json:"original_resume_id,omitempty"`
	TailoredFileID   string        `json:"tailored_file_id,omitempty"`
	DownloadPath     string        `json:"download_path,omitempty"`
	KeywordsAdded    []string      `json:"keywords_added"`
	ChangesSummary   string        `json:"changes_summary"`
}

func (h *Handler) anyAI() bool {
	return (h.gemini != nil && h.gemini.Enabled()) || (h.groq != nil && h.groq.Enabled())
}

func (h *Handler) requireAnyAI(c *gin.Context) bool {
	if !h.anyAI() {
		response.BadRequest(c, "AI is not configured (set GEMINI_API_KEY and/or GROQ_API_KEY)")
		return false
	}
	return true
}

func (h *Handler) chatSmall(system, user string, maxTokens int) (string, string, error) {
	if h.gemini != nil && h.gemini.Enabled() {
		text, err := h.gemini.Chat(system, user, maxTokens)
		if err == nil {
			return text, "gemini/" + h.gemini.Model(), nil
		}
		if h.groq == nil || !h.groq.Enabled() {
			return "", "", err
		}
	}
	if h.groq != nil && h.groq.Enabled() {
		text, err := h.groq.Chat(system, user, maxTokens)
		if err != nil {
			return "", "", err
		}
		return text, "groq/" + h.groq.Model(), nil
	}
	return "", "", fmt.Errorf("no AI provider configured")
}

func (h *Handler) loadJobAndUser(c *gin.Context, req *jobContextRequest) (*models.User, string, string, string, bool) {
	user, err := h.auth.GetByID(middleware.UserID(c))
	if err != nil {
		response.NotFound(c, "user not found")
		return nil, "", "", "", false
	}
	title := strings.TrimSpace(req.Title)
	company := strings.TrimSpace(req.Company)
	description := strings.TrimSpace(req.Description)
	if id := strings.TrimSpace(req.JobID); id != "" {
		if jobUUID, err := uuid.Parse(id); err == nil {
			var job models.Job
			if err := h.db.First(&job, "id = ?", jobUUID).Error; err == nil {
				if title == "" {
					title = job.Title
				}
				if company == "" {
					company = job.Company
				}
				if description == "" {
					description = job.Description
				}
			}
		}
	}
	if title == "" || company == "" {
		response.BadRequest(c, "job title and company are required")
		return nil, "", "", "", false
	}
	return user, title, company, description, true
}

func (h *Handler) latestResume(userID uuid.UUID) (*models.Resume, error) {
	var resume models.Resume
	if err := h.db.Where("user_id = ?", userID).Order("created_at desc").First(&resume).Error; err != nil {
		return nil, err
	}
	return &resume, nil
}

func (h *Handler) latestResumeText(userID uuid.UUID) string {
	resume, err := h.latestResume(userID)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(resume.ParsedText)
}

func profileBlob(user *models.User, resumeText string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Name: %s\n", user.Name)
	if user.Headline != "" {
		fmt.Fprintf(&b, "Headline: %s\n", user.Headline)
	}
	if user.Location != "" {
		fmt.Fprintf(&b, "Location: %s\n", user.Location)
	}
	if user.Skills != "" {
		fmt.Fprintf(&b, "Skills: %s\n", user.Skills)
	}
	if user.Education != "" {
		fmt.Fprintf(&b, "Education:\n%s\n", trim(user.Education, 1000))
	}
	if user.LinkedIn != "" {
		fmt.Fprintf(&b, "LinkedIn: %s\n", user.LinkedIn)
	}
	if user.GitHub != "" {
		fmt.Fprintf(&b, "GitHub: %s\n", user.GitHub)
	}
	if user.Website != "" {
		fmt.Fprintf(&b, "Website: %s\n", user.Website)
	}
	if user.Phone != "" {
		fmt.Fprintf(&b, "Phone: %s\n", user.Phone)
	}
	fmt.Fprintf(&b, "Email: %s\n", user.Email)
	if resumeText != "" {
		fmt.Fprintf(&b, "\n--- UPLOADED RESUME TEXT ---\n%s\n", trim(resumeText, 5500))
	}
	return b.String()
}

func (h *Handler) Analyze(c *gin.Context) {
	var req jobContextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid payload")
		return
	}
	user, title, company, description, ok := h.loadJobAndUser(c, &req)
	if !ok {
		return
	}
	result, err := h.runAnalyze(user, title, company, description)
	if err != nil {
		response.Internal(c, err.Error())
		return
	}
	response.OK(c, result)
}

func (h *Handler) runAnalyze(user *models.User, title, company, description string) (*analyzeResult, error) {
	resumeText := h.latestResumeText(user.ID)
	if resumeText == "" {
		resumeText = profileBlob(user, "")
	}
	base := atsscore.Score(resumeText+"\n"+user.Skills+"\n"+user.Headline, title, description)

	out := &analyzeResult{
		MatchScore:      base.MatchScore,
		MissingKeywords: base.MissingKeywords,
		MissingSkills:   base.MissingSkills,
		Strengths:       base.Present,
		Covered:         base.Covered,
		TotalKeywords:   base.Total,
		Model:           "keyword-coverage",
		Summary: fmt.Sprintf(
			"Keyword coverage %d/%d (%.0f%%). Score is based on how many job keywords appear in your uploaded resume — not a model guess.",
			base.Covered, base.Total, base.MatchScore,
		),
		Suggestions: []string{
			"Keep your 1-page PDF; we weave missing keywords into your Skills and Projects when you tailor.",
			"Only claim keywords you can defend in interviews.",
		},
	}

	// Optional AI narrative on top of deterministic gaps (does not override score).
	if h.anyAI() {
		system := `You advise on ATS gaps. Return ONLY JSON:
{"suggestions":[],"summary":"","strengths_note":""}
Use the provided coverage numbers. Do not invent a new match_score. Max 5 suggestions.`
		userPrompt := fmt.Sprintf(
			"Role: %s at %s\nCoverage: %d/%d = %.1f%%\nPresent: %s\nMissing skills: %s\nMissing keywords: %s\n\nResume excerpt:\n%s",
			title, company, base.Covered, base.Total, base.MatchScore,
			strings.Join(base.Present, ", "),
			strings.Join(base.MissingSkills, ", "),
			strings.Join(base.MissingKeywords, ", "),
			trim(resumeText, 1800),
		)
		raw, model, err := h.chatSmall(system, userPrompt, 450)
		if err == nil {
			var narr struct {
				Suggestions   []string `json:"suggestions"`
				Summary       string   `json:"summary"`
				StrengthsNote string   `json:"strengths_note"`
			}
			if json.Unmarshal([]byte(extractJSONObject(raw)), &narr) == nil {
				if narr.Summary != "" {
					out.Summary = narr.Summary + fmt.Sprintf(" (coverage %.0f%% · %d/%d keywords)", base.MatchScore, base.Covered, base.Total)
				}
				if len(narr.Suggestions) > 0 {
					out.Suggestions = narr.Suggestions
				}
				out.Model = "keyword-coverage+" + model
			}
		}
	}
	return out, nil
}

func (h *Handler) Tailor(c *gin.Context) {
	var req jobContextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid payload")
		return
	}
	user, title, company, description, ok := h.loadJobAndUser(c, &req)
	if !ok {
		return
	}

	analyze := analyzeResult{
		MissingKeywords: req.MissingKeywords,
		MissingSkills:   req.MissingSkills,
		Suggestions:     req.Suggestions,
	}
	if len(analyze.MissingKeywords) == 0 && len(analyze.MissingSkills) == 0 {
		a, err := h.runAnalyze(user, title, company, description)
		if err != nil {
			response.Internal(c, err.Error())
			return
		}
		analyze = *a
	} else {
		// Recompute authoritative score even if client sent gaps.
		a, err := h.runAnalyze(user, title, company, description)
		if err == nil {
			analyze.MatchScore = a.MatchScore
			analyze.Strengths = a.Strengths
			analyze.Covered = a.Covered
			analyze.TotalKeywords = a.TotalKeywords
			analyze.Summary = a.Summary
			analyze.Model = a.Model
			if len(analyze.Suggestions) == 0 {
				analyze.Suggestions = a.Suggestions
			}
		}
	}

	out, err := h.runTailor(user, req.JobID, title, company, description, req.Tone, req.Extra, &analyze)
	if err != nil {
		response.Internal(c, err.Error())
		return
	}
	response.OK(c, out)
}

func (h *Handler) Prepare(c *gin.Context) {
	var req jobContextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid payload")
		return
	}
	user, title, company, description, ok := h.loadJobAndUser(c, &req)
	if !ok {
		return
	}
	analyze, err := h.runAnalyze(user, title, company, description)
	if err != nil {
		response.Internal(c, err.Error())
		return
	}
	out, err := h.runTailor(user, req.JobID, title, company, description, req.Tone, req.Extra, analyze)
	if err != nil {
		response.Internal(c, err.Error())
		return
	}
	response.OK(c, out)
}

func (h *Handler) runTailor(user *models.User, jobIDStr, title, company, description, tone, extra string, analyze *analyzeResult) (*tailorResult, error) {
	tone = strings.ToLower(strings.TrimSpace(tone))
	if tone != "concise" && tone != "enthusiastic" {
		tone = "professional"
	}

	resume, err := h.latestResume(user.ID)
	if err != nil {
		return nil, fmt.Errorf("upload a resume PDF on Profile first")
	}
	if !strings.EqualFold(filepath.Ext(resume.FilePath), ".pdf") && !strings.Contains(strings.ToLower(resume.ContentType), "pdf") {
		return nil, fmt.Errorf("tailoring needs a PDF resume — re-upload as PDF")
	}
	if _, err := os.Stat(resume.FilePath); err != nil {
		return nil, fmt.Errorf("original resume file missing on disk")
	}

	// Ask AI which missing keywords can honestly be stamped (short list). No full rewrite.
	candidates := append([]string{}, analyze.MissingSkills...)
	candidates = append(candidates, analyze.MissingKeywords...)
	keywords := pickHonestKeywords(candidates, 8)
	if h.anyAI() && len(candidates) > 0 {
		system := `Pick keywords from the candidate list that honestly fit this resume. Return ONLY JSON:
{"keywords_to_add":[],"note":""}
Max 8 keywords. Never invent employers or degrees. Prefer skills already implied by the resume.`
		userPrompt := fmt.Sprintf("CANDIDATES: %s\n\nRESUME:\n%s\n\nJOB: %s at %s\n%s",
			strings.Join(candidates, ", "),
			trim(resume.ParsedText, 3500),
			title, company, trim(description, 800),
		)
		raw, _, err := h.chatSmall(system, userPrompt, 300)
		if err == nil {
			var pick struct {
				Keywords []string `json:"keywords_to_add"`
			}
			if json.Unmarshal([]byte(extractJSONObject(raw)), &pick) == nil && len(pick.Keywords) > 0 {
				keywords = pickHonestKeywords(pick.Keywords, 8)
			}
		}
	}

	fileID := uuid.New().String()
	outDir := filepath.Join(h.uploadDir, "tailored", user.ID.String())
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}
	dst := filepath.Join(outDir, fileID+".pdf")
	stampTitle := fmt.Sprintf("%s @ %s", title, company)
	if err := pdfstamp.Stamp(h.pythonBin, h.stampScript, resume.FilePath, dst, stampTitle, keywords); err != nil {
		return nil, fmt.Errorf("could not update PDF: %w", err)
	}

	coverLetter := ""
	model := "pdf-stamp"
	if h.anyAI() {
		coverSystem := `Write a job application cover letter. Return ONLY the letter body — no markdown fences. Under 180 words.`
		coverPrompt := fmt.Sprintf("Tone: %s\nName: %s\nRole: %s at %s\nKeywords emphasized: %s\nJD:\n%s\n%s",
			tone, user.Name, title, company, strings.Join(keywords, ", "), trim(description, 1400), extraLine(extra))
		letter, m, err := h.chatSmall(coverSystem, coverPrompt, 450)
		if err == nil {
			coverLetter = cleanPlainResume(letter)
			model = m + "+pdf-stamp"
			_ = h.db.Model(&models.User{}).Where("id = ?", user.ID).Update("cover_letter", coverLetter).Error
		}
	}

	summary := fmt.Sprintf(
		"Updated your original PDF for %s — keywords woven into Technical Skills and Projects (same layout/fonts).",
		stampTitle,
	)
	preview := summary + "\n\nKeywords added:\n- " + strings.Join(keywords, "\n- ")
	if len(keywords) == 0 {
		preview = summary + "\n\nNo extra keywords needed — your resume already covers the main terms."
	}

	// Text used for future ATS compare = original parse + added keywords.
	tailoredText := strings.TrimSpace(resume.ParsedText + "\n" + strings.Join(keywords, " "))
	after := atsscore.Score(tailoredText+"\n"+user.Skills+"\n"+user.Headline, title, description)

	var jobUUID *uuid.UUID
	if id, err := uuid.Parse(strings.TrimSpace(jobIDStr)); err == nil {
		jobUUID = &id
	}
	srcID := resume.ID
	rec := &models.TailoredVersion{
		UserID:          user.ID,
		JobID:           jobUUID,
		JobTitle:        title,
		JobCompany:      company,
		MatchScore:      analyze.MatchScore,
		AfterScore:      after.MatchScore,
		Covered:         after.Covered,
		TotalKeywords:   after.Total,
		KeywordsAdded:   keywords,
		MissingKeywords: after.MissingKeywords,
		MissingSkills:   after.MissingSkills,
		FileID:          fileID,
		FilePath:        dst,
		ParsedText:      tailoredText,
		CoverLetter:     coverLetter,
		SourceResumeID:  &srcID,
	}
	_ = h.db.Create(rec).Error

	return &tailorResult{
		Headline:         firstNonEmpty(user.Headline, title),
		Summary:          summary,
		Skills:           keywords,
		ResumeMarkdown:   preview,
		CoverLetter:      coverLetter,
		Analyze:          *analyze,
		Model:            model,
		OriginalResumeID: resume.ID.String(),
		TailoredFileID:   fileID,
		DownloadPath:     "/api/v1/ai/tailored/" + fileID + "/file",
		KeywordsAdded:    keywords,
		ChangesSummary:   summary,
	}, nil
}

type tailoredListItem struct {
	ID                 string   `json:"id"`
	FileID             string   `json:"file_id"`
	JobID              string   `json:"job_id,omitempty"`
	JobTitle           string   `json:"job_title"`
	JobCompany         string   `json:"job_company"`
	MatchScore         float64  `json:"match_score"`
	AfterScore         float64  `json:"after_score"`
	ScoreForCurrentJob float64  `json:"score_for_current_job"`
	Covered            int      `json:"covered"`
	TotalKeywords      int      `json:"total_keywords"`
	KeywordsAdded      []string `json:"keywords_added"`
	MissingKeywords    []string `json:"missing_keywords"`
	MissingSkills      []string `json:"missing_skills"`
	DownloadPath       string   `json:"download_path"`
	CoverLetter        string   `json:"cover_letter,omitempty"`
	CreatedAt          string   `json:"created_at"`
	ForThisJob         bool     `json:"for_this_job"`
}

func (h *Handler) ListTailored(c *gin.Context) {
	userID := middleware.UserID(c)
	title := strings.TrimSpace(c.Query("title"))
	company := strings.TrimSpace(c.Query("company"))
	description := strings.TrimSpace(c.Query("description"))
	jobID := strings.TrimSpace(c.Query("job_id"))

	var rows []models.TailoredVersion
	if err := h.db.Where("user_id = ?", userID).Order("created_at desc").Limit(40).Find(&rows).Error; err != nil {
		response.Internal(c, "failed to list tailored resumes")
		return
	}

	out := make([]tailoredListItem, 0, len(rows))
	for _, r := range rows {
		scoreCurrent := r.AfterScore
		covered, total := r.Covered, r.TotalKeywords
		missingK, missingS := r.MissingKeywords, r.MissingSkills
		if title != "" || description != "" {
			sc := atsscore.Score(r.ParsedText, title, description)
			scoreCurrent = sc.MatchScore
			covered, total = sc.Covered, sc.Total
			missingK, missingS = sc.MissingKeywords, sc.MissingSkills
		}
		jid := ""
		if r.JobID != nil {
			jid = r.JobID.String()
		}
		forThis := false
		if jobID != "" && jid == jobID {
			forThis = true
		} else if jobID == "" && title != "" && company != "" {
			forThis = strings.EqualFold(r.JobTitle, title) && strings.EqualFold(r.JobCompany, company)
		}
		out = append(out, tailoredListItem{
			ID:                 r.ID.String(),
			FileID:             r.FileID,
			JobID:              jid,
			JobTitle:           r.JobTitle,
			JobCompany:         r.JobCompany,
			MatchScore:         r.MatchScore,
			AfterScore:         r.AfterScore,
			ScoreForCurrentJob: scoreCurrent,
			Covered:            covered,
			TotalKeywords:      total,
			KeywordsAdded:      r.KeywordsAdded,
			MissingKeywords:    missingK,
			MissingSkills:      missingS,
			DownloadPath:       "/api/v1/ai/tailored/" + r.FileID + "/file",
			CoverLetter:        r.CoverLetter,
			CreatedAt:          r.CreatedAt.UTC().Format(time.RFC3339),
			ForThisJob:         forThis,
		})
	}

	// Best first for current job compare
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].ScoreForCurrentJob == out[j].ScoreForCurrentJob {
			return out[i].CreatedAt > out[j].CreatedAt
		}
		return out[i].ScoreForCurrentJob > out[j].ScoreForCurrentJob
	})

	response.OK(c, gin.H{"items": out, "count": len(out)})
}

func (h *Handler) DownloadTailored(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" || strings.Contains(id, "..") || strings.Contains(id, "/") {
		response.BadRequest(c, "invalid file id")
		return
	}
	userID := middleware.UserID(c)
	path := filepath.Join(h.uploadDir, "tailored", userID.String(), id+".pdf")
	if _, err := os.Stat(path); err != nil {
		response.NotFound(c, "tailored PDF not found")
		return
	}
	c.FileAttachment(path, "tailored-resume.pdf")
}

func (h *Handler) DownloadOriginal(c *gin.Context) {
	userID := middleware.UserID(c)
	resume, err := h.latestResume(userID)
	if err != nil {
		response.NotFound(c, "resume not found")
		return
	}
	c.FileAttachment(resume.FilePath, resume.FileName)
}

func (h *Handler) CoverLetter(c *gin.Context) {
	if !h.requireAnyAI(c) {
		return
	}
	var req jobContextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid payload")
		return
	}
	user, title, company, description, ok := h.loadJobAndUser(c, &req)
	if !ok {
		return
	}
	tone := strings.ToLower(strings.TrimSpace(req.Tone))
	if tone != "concise" && tone != "enthusiastic" {
		tone = "professional"
	}
	resumeText := h.latestResumeText(user.ID)
	system := `Write a job application cover letter. Return ONLY the letter body — no markdown fences, no title. Under 180 words.`
	userPrompt := fmt.Sprintf("Tone: %s\n\nCandidate:\n%s\n\nRole: %s at %s\nJD:\n%s\n%s",
		tone, profileBlob(user, resumeText), title, company, trim(description, 1800), extraLine(req.Extra))
	letter, model, err := h.chatSmall(system, userPrompt, 450)
	if err != nil {
		response.Internal(c, err.Error())
		return
	}
	letter = cleanPlainResume(letter)
	_ = h.db.Model(&models.User{}).Where("id = ?", user.ID).Update("cover_letter", letter).Error
	response.OK(c, gin.H{"cover_letter": letter, "tone": tone, "model": model})
}

func (h *Handler) Status(c *gin.Context) {
	geminiOn := h.gemini != nil && h.gemini.Enabled()
	groqOn := h.groq != nil && h.groq.Enabled()
	geminiModel := ""
	groqModel := ""
	if geminiOn {
		geminiModel = h.gemini.Model()
	}
	if groqOn {
		groqModel = h.groq.Model()
	}
	response.OK(c, gin.H{
		"enabled": geminiOn || groqOn,
		"routing": gin.H{
			"ats_score":     "deterministic keyword coverage (+ optional AI notes)",
			"resume_tailor": "weave keywords into skills/projects on original PDF",
			"cover_letter":  "gemini (fallback groq)",
		},
		"pdf_stamp": gin.H{
			"python": h.pythonBin,
			"script": h.stampScript,
		},
		"gemini": gin.H{"enabled": geminiOn, "model": geminiModel},
		"groq":   gin.H{"enabled": groqOn, "model": groqModel},
	})
}

func pickHonestKeywords(in []string, max int) []string {
	seen := map[string]bool{}
	out := make([]string, 0, max)
	for _, k := range in {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		key := strings.ToLower(k)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, k)
		if len(out) >= max {
			break
		}
	}
	return out
}

func resolveStampTools() (python, script string) {
	if v := strings.TrimSpace(os.Getenv("PDF_PYTHON")); v != "" {
		if st, err := os.Stat(v); err == nil && !st.IsDir() {
			python = v
		} else if p, err := exec.LookPath(v); err == nil {
			python = p
		}
	}
	if v := strings.TrimSpace(os.Getenv("STAMP_PDF_SCRIPT")); v != "" {
		if st, err := os.Stat(v); err == nil && !st.IsDir() {
			script = v
		}
	}

	roots := []string{".", "..", "../..", "../../..", "/app"}
	if python == "" {
		candidates := []string{
			"/app/.venv-pdf/bin/python",
			"/app/.venv-pdf/bin/python3",
			".venv-pdf/bin/python",
			".venv-pdf/bin/python3",
		}
		for _, r := range roots {
			candidates = append(candidates, filepath.Join(r, ".venv-pdf", "bin", "python"))
		}
		for _, p := range candidates {
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				python, _ = filepath.Abs(p)
				break
			}
		}
	}
	if python == "" {
		for _, name := range []string{"python3", "python"} {
			if p, err := exec.LookPath(name); err == nil {
				python = p
				break
			}
		}
	}
	if python == "" {
		python = "python3"
	}

	if script == "" {
		for _, p := range []string{
			"/app/scripts/stamp_pdf.py",
			"scripts/stamp_pdf.py",
			"services/api/scripts/stamp_pdf.py",
		} {
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				script, _ = filepath.Abs(p)
				break
			}
		}
		if script == "" {
			for _, r := range roots {
				p := filepath.Join(r, "services", "api", "scripts", "stamp_pdf.py")
				if st, err := os.Stat(p); err == nil && !st.IsDir() {
					script, _ = filepath.Abs(p)
					break
				}
			}
		}
	}
	if script == "" {
		script = "/app/scripts/stamp_pdf.py"
	}
	return python, script
}

func stripJSONFence(s string) string {
	s = strings.TrimSpace(s)
	for {
		prev := s
		if strings.HasPrefix(s, "```") {
			if i := strings.Index(s, "\n"); i >= 0 {
				s = s[i+1:]
			} else {
				s = strings.TrimPrefix(s, "```")
			}
		}
		s = strings.TrimSpace(s)
		if strings.HasSuffix(s, "```") {
			s = strings.TrimSpace(strings.TrimSuffix(s, "```"))
		}
		if s == prev {
			break
		}
	}
	return strings.TrimSpace(s)
}

func cleanPlainResume(s string) string {
	return stripJSONFence(s)
}

func extractJSONObject(s string) string {
	s = stripJSONFence(s)
	start := strings.Index(s, "{")
	if start < 0 {
		return s
	}
	depth := 0
	inString := false
	escape := false
	for i := start; i < len(s); i++ {
		ch := s[i]
		if inString {
			if escape {
				escape = false
				continue
			}
			if ch == '\\' {
				escape = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}
		switch ch {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return strings.TrimSpace(s[start : i+1])
			}
		}
	}
	return strings.TrimSpace(s[start:])
}

func splitCSVSkills(s string) []string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n'
	})
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
		if len(out) >= 18 {
			break
		}
	}
	return out
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func extraLine(extra string) string {
	extra = strings.TrimSpace(extra)
	if extra == "" {
		return ""
	}
	return "\nExtra instructions: " + extra
}

func trim(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.GET("/status", h.Status)
	rg.POST("/analyze", h.Analyze)
	rg.POST("/tailor", h.Tailor)
	rg.POST("/prepare", h.Prepare)
	rg.POST("/cover-letter", h.CoverLetter)
	rg.GET("/tailored", h.ListTailored)
	rg.GET("/tailored/:id/file", h.DownloadTailored)
	rg.GET("/original-resume/file", h.DownloadOriginal)
}
