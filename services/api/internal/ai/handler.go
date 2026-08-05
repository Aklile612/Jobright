package ai

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jobright/api/internal/auth"
	"github.com/jobright/api/internal/gemini"
	"github.com/jobright/api/internal/middleware"
	"github.com/jobright/api/internal/models"
	"github.com/jobright/api/pkg/response"
	"gorm.io/gorm"
)

type Handler struct {
	db     *gorm.DB
	auth   *auth.Service
	gemini *gemini.Client
}

func NewHandler(db *gorm.DB, authSvc *auth.Service, geminiClient *gemini.Client) *Handler {
	return &Handler{db: db, auth: authSvc, gemini: geminiClient}
}

type coverLetterRequest struct {
	JobID    string `json:"job_id"`
	Title    string `json:"title"`
	Company  string `json:"company"`
	Description string `json:"description"`
	Tone     string `json:"tone"` // professional | concise | enthusiastic
	Extra    string `json:"extra"`
}

func (h *Handler) CoverLetter(c *gin.Context) {
	if h.gemini == nil || !h.gemini.Enabled() {
		response.BadRequest(c, "AI cover letter is not configured (set GEMINI_API_KEY)")
		return
	}

	var req coverLetterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid payload")
		return
	}

	user, err := h.auth.GetByID(middleware.UserID(c))
	if err != nil {
		response.NotFound(c, "user not found")
		return
	}

	title := strings.TrimSpace(req.Title)
	company := strings.TrimSpace(req.Company)
	description := strings.TrimSpace(req.Description)

	if id := strings.TrimSpace(req.JobID); id != "" {
		jobUUID, err := uuid.Parse(id)
		if err == nil {
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
		return
	}

	tone := strings.ToLower(strings.TrimSpace(req.Tone))
	switch tone {
	case "concise", "enthusiastic", "professional":
	default:
		tone = "professional"
	}

	prompt := buildCoverLetterPrompt(user, title, company, description, tone, strings.TrimSpace(req.Extra))
	letter, err := h.gemini.GenerateCoverLetter(prompt)
	if err != nil {
		response.Internal(c, err.Error())
		return
	}

	// Persist as default cover letter draft for autofill next time.
	_ = h.db.Model(&models.User{}).Where("id = ?", user.ID).Update("cover_letter", letter).Error

	response.OK(c, gin.H{
		"cover_letter": letter,
		"tone":         tone,
	})
}

func buildCoverLetterPrompt(user *models.User, title, company, description, tone, extra string) string {
	desc := description
	if len(desc) > 4500 {
		desc = desc[:4500] + "…"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Write a %s job application cover letter.\n", tone)
	b.WriteString("Return ONLY the letter body — no markdown fences, no title, no commentary.\n")
	b.WriteString("Keep it under 280 words. Use first person. Be specific to the role.\n")
	b.WriteString("Do not invent employers or degrees the candidate did not list.\n\n")
	fmt.Fprintf(&b, "Candidate name: %s\n", nonEmpty(user.Name, "Applicant"))
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
		fmt.Fprintf(&b, "Education: %s\n", trim(user.Education, 800))
	}
	if user.CoverLetter != "" {
		fmt.Fprintf(&b, "Existing summary / experience notes:\n%s\n", trim(user.CoverLetter, 1200))
	}
	fmt.Fprintf(&b, "\nTarget role: %s\nCompany: %s\n", title, company)
	fmt.Fprintf(&b, "Job description:\n%s\n", desc)
	if extra != "" {
		fmt.Fprintf(&b, "\nExtra instructions from candidate: %s\n", extra)
	}
	return b.String()
}

func nonEmpty(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return strings.TrimSpace(v)
}

func trim(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.POST("/cover-letter", h.CoverLetter)
}
