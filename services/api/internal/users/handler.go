package users

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jobright/api/internal/auth"
	"github.com/jobright/api/internal/middleware"
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
	Name string `json:"name"`
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
	user.Name = strings.TrimSpace(req.Name)
	if err := h.db.Model(user).Update("name", user.Name).Error; err != nil {
		response.Internal(c, "failed to update profile")
		return
	}
	response.OK(c, gin.H{
		"id":                user.ID,
		"email":             user.Email,
		"name":              user.Name,
		"current_resume_id": user.CurrentResumeID,
	})
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.PATCH("/me", h.UpdateProfile)
}
