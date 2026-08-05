package users

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jobright/api/internal/auth"
	"github.com/jobright/api/internal/middleware"
	"github.com/jobright/api/internal/models"
	"github.com/jobright/api/pkg/response"
	"gorm.io/gorm"
)

type Handler struct {
	db   *gorm.DB
	auth *auth.Service
}

func NewHandler(db *gorm.DB, authSvc *auth.Service) *Handler {
	return &Handler{db: db, auth: authSvc}
}

type updateProfileRequest struct {
	Name        string `json:"name"`
	Phone       string `json:"phone"`
	LinkedIn    string `json:"linkedin"`
	GitHub      string `json:"github"`
	Website     string `json:"website"`
	Location    string `json:"location"`
	Headline    string `json:"headline"`
	CoverLetter string `json:"cover_letter"`
}

func toProfile(user *models.User) gin.H {
	return gin.H{
		"id":                user.ID,
		"email":             user.Email,
		"name":              user.Name,
		"phone":             user.Phone,
		"linkedin":          user.LinkedIn,
		"github":            user.GitHub,
		"website":           user.Website,
		"location":          user.Location,
		"headline":          user.Headline,
		"cover_letter":      user.CoverLetter,
		"current_resume_id": user.CurrentResumeID,
	}
}

func (h *Handler) Me(c *gin.Context) {
	user, err := h.auth.GetByID(middleware.UserID(c))
	if err != nil {
		response.NotFound(c, "user not found")
		return
	}
	response.OK(c, toProfile(user))
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	var req updateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid profile payload")
		return
	}
	user, err := h.auth.GetByID(middleware.UserID(c))
	if err != nil {
		response.NotFound(c, "user not found")
		return
	}
	updates := map[string]any{
		"name":         strings.TrimSpace(req.Name),
		"phone":        strings.TrimSpace(req.Phone),
		"linkedin":     strings.TrimSpace(req.LinkedIn),
		"github":       strings.TrimSpace(req.GitHub),
		"website":      strings.TrimSpace(req.Website),
		"location":     strings.TrimSpace(req.Location),
		"headline":     strings.TrimSpace(req.Headline),
		"cover_letter": strings.TrimSpace(req.CoverLetter),
	}
	if err := h.db.Model(user).Updates(updates).Error; err != nil {
		response.Internal(c, "failed to update profile")
		return
	}
	user, err = h.auth.GetByID(user.ID)
	if err != nil {
		response.Internal(c, "failed to reload profile")
		return
	}
	response.OK(c, toProfile(user))
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.GET("/me", h.Me)
	rg.PATCH("/me", h.UpdateProfile)
}
