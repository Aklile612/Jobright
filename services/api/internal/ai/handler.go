package ai

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jobright/api/internal/auth"
	"github.com/jobright/api/internal/groq"
	"github.com/jobright/api/internal/middleware"
	"github.com/jobright/api/internal/models"
	"github.com/jobright/api/pkg/response"
	"gorm.io/gorm"
)

type Handler struct {
	db   *gorm.DB
	auth *auth.Service
	ai   *groq.Client
}

func NewHandler(db *gorm.DB, authSvc *auth.Service, aiClient *groq.Client) *Handler {
	return &Handler{db: db, auth: authSvc, ai: aiClient}
}

type jobContextRequest struct {
	JobID       string `json:"job_id"`
	Title       string `json:"title"`
	Company     string `json:"company"`
	Description string `json:"description"`
	Tone        string `json:"tone"`
	Extra       string `json:"extra"`
}

type analyzeResult struct {
	MatchScore       float64  `json:"match_score"`
	MissingKeywords  []string `json:"missing_keywords"`
	MissingSkills    []string `json:"missing_skills"`
	Strengths        []string `json:"strengths"`
	Suggestions      []string `json:"suggestions"`
	Summary          string   `json:"summary"`
	Model            string   `json:"model"`
}

type tailorResult struct {
	Headline        string   `json:"headline"`
	Summary         string   `json:"summary"`
	Skills          []string `json:"skills"`
	ExperienceBullets []string `json:"experience_bullets"`
	Education       string   `json:"education"`
	ResumeMarkdown  string   `json:"resume_markdown"`
	CoverLetter     string   `json:"cover_letter"`
	Analyze         analyzeResult `json:"analyze"`
	Model           string   `json:"model"`
}

func (h *Handler) requireAI(c *gin.Context) bool {
	if h.ai == nil || !h.ai.Enabled() {
		response.BadRequest(c, "AI is not configured (set GROQ_API_KEY)")
		return false
	}
	return true
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

func profileBlob(user *models.User) string {
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
		fmt.Fprintf(&b, "Education:\n%s\n", trim(user.Education, 1200))
	}
	if user.CoverLetter != "" {
		fmt.Fprintf(&b, "Experience / summary notes:\n%s\n", trim(user.CoverLetter, 2500))
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
	return b.String()
}

func (h *Handler) Analyze(c *gin.Context) {
	if !h.requireAI(c) {
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
	result, err := h.runAnalyze(user, title, company, description)
	if err != nil {
		response.Internal(c, err.Error())
		return
	}
	response.OK(c, result)
}

func (h *Handler) runAnalyze(user *models.User, title, company, description string) (*analyzeResult, error) {
	system := `You are an ATS resume analyst. Compare the candidate profile to the job.
Return ONLY valid JSON with this shape:
{"match_score":0-100,"missing_keywords":[],"missing_skills":[],"strengths":[],"suggestions":[],"summary":""}
Do not invent degrees or employers the candidate did not list.
Keep arrays to at most 8 items. summary max 2 sentences.`
	userPrompt := fmt.Sprintf("CANDIDATE PROFILE:\n%s\n\nJOB:\nTitle: %s\nCompany: %s\nDescription:\n%s",
		profileBlob(user), title, company, trim(description, 5000))
	raw, err := h.ai.Chat(system, userPrompt, 1200)
	if err != nil {
		return nil, err
	}
	raw = stripJSONFence(raw)
	var out analyzeResult
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("failed to parse ATS analysis: %w", err)
	}
	out.Model = h.ai.Model()
	if out.MatchScore < 0 {
		out.MatchScore = 0
	}
	if out.MatchScore > 100 {
		out.MatchScore = 100
	}
	return &out, nil
}

func (h *Handler) Prepare(c *gin.Context) {
	if !h.requireAI(c) {
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

	analyze, err := h.runAnalyze(user, title, company, description)
	if err != nil {
		response.Internal(c, err.Error())
		return
	}

	tone := strings.ToLower(strings.TrimSpace(req.Tone))
	if tone != "concise" && tone != "enthusiastic" {
		tone = "professional"
	}

	system := `You tailor an existing resume for one job. Do NOT invent fake employers, degrees, or dates.
Only weave in missing keywords/skills honestly from the candidate's real background (rephrase experience, add skill terms they plausibly have).
Return ONLY valid JSON:
{"headline":"","summary":"","skills":[],"experience_bullets":[],"education":"","resume_markdown":"","cover_letter":""}
- skills: merged list (existing + relevant missing terms), max 25
- experience_bullets: 4-8 bullets grounded in their notes
- resume_markdown: full plain resume in markdown they can copy
- cover_letter: under 280 words, ` + tone + ` tone, no markdown fences`

	userPrompt := fmt.Sprintf(`CANDIDATE PROFILE:
%s

ATS GAPS:
missing_skills=%s
missing_keywords=%s
suggestions=%s

TARGET JOB:
Title: %s
Company: %s
Description:
%s
%s`,
		profileBlob(user),
		strings.Join(analyze.MissingSkills, ", "),
		strings.Join(analyze.MissingKeywords, ", "),
		strings.Join(analyze.Suggestions, "; "),
		title, company, trim(description, 4500),
		extraLine(req.Extra),
	)

	raw, err := h.ai.Chat(system, userPrompt, 3500)
	if err != nil {
		response.Internal(c, err.Error())
		return
	}
	raw = stripJSONFence(raw)
	var tailored tailorResult
	if err := json.Unmarshal([]byte(raw), &tailored); err != nil {
		response.Internal(c, "failed to parse tailored resume: "+err.Error())
		return
	}
	tailored.Analyze = *analyze
	tailored.Model = h.ai.Model()
	if tailored.CoverLetter != "" {
		_ = h.db.Model(&models.User{}).Where("id = ?", user.ID).Update("cover_letter", tailored.CoverLetter).Error
	}
	response.OK(c, tailored)
}

func (h *Handler) CoverLetter(c *gin.Context) {
	if !h.requireAI(c) {
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
	system := `Write a job application cover letter. Return ONLY the letter body — no markdown, no title.`
	userPrompt := fmt.Sprintf("Tone: %s\n\nCandidate:\n%s\n\nRole: %s at %s\nJD:\n%s\n%s",
		tone, profileBlob(user), title, company, trim(description, 4000), extraLine(req.Extra))
	letter, err := h.ai.Chat(system, userPrompt, 900)
	if err != nil {
		response.Internal(c, err.Error())
		return
	}
	_ = h.db.Model(&models.User{}).Where("id = ?", user.ID).Update("cover_letter", letter).Error
	response.OK(c, gin.H{"cover_letter": letter, "tone": tone, "model": h.ai.Model()})
}

func (h *Handler) Status(c *gin.Context) {
	enabled := h.ai != nil && h.ai.Enabled()
	model := ""
	if enabled {
		model = h.ai.Model()
	}
	response.OK(c, gin.H{
		"enabled": enabled,
		"provider": "groq",
		"model": model,
		"recommended": gin.H{
			"volume":  "llama-3.1-8b-instant",
			"quality": "llama-3.3-70b-versatile",
			"note":    "instant ≈ 14.4k req/day + 500k tokens/day on free tier; 70b ≈ 1k req/day + 100k tokens/day",
		},
	})
}

func stripJSONFence(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```JSON")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
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
	rg.POST("/prepare", h.Prepare)
	rg.POST("/cover-letter", h.CoverLetter)
}
