package auth

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jobright/api/internal/middleware"
	"github.com/jobright/api/internal/models"
	"github.com/jobright/api/pkg/response"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

type credentialsRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name"`
}

type authResponse struct {
	Token string       `json:"token"`
	User  userResponse `json:"user"`
}

type userResponse struct {
	ID              uuid.UUID  `json:"id"`
	Email           string     `json:"email"`
	Name            string     `json:"name"`
	CurrentResumeID *uuid.UUID `json:"current_resume_id,omitempty"`
}

func (h *Handler) Register(c *gin.Context) {
	var req credentialsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid signup payload")
		return
	}
	user, token, err := h.svc.Signup(strings.TrimSpace(strings.ToLower(req.Email)), req.Password, strings.TrimSpace(req.Name))
	if err != nil {
		if errors.Is(err, ErrEmailTaken) {
			response.Conflict(c, "email already registered")
			return
		}
		response.Internal(c, err.Error())
		return
	}
	response.Created(c, authResponse{Token: token, User: toUser(user)})
}

func (h *Handler) Login(c *gin.Context) {
	var req credentialsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid login payload")
		return
	}
	user, token, err := h.svc.Login(strings.TrimSpace(strings.ToLower(req.Email)), req.Password)
	if err != nil {
		response.Unauthorized(c, "invalid email or password")
		return
	}
	response.OK(c, authResponse{Token: token, User: toUser(user)})
}

func (h *Handler) Me(c *gin.Context) {
	user, err := h.svc.GetByID(middleware.UserID(c))
	if err != nil {
		response.NotFound(c, "user not found")
		return
	}
	response.OK(c, toUser(user))
}

func toUser(u *models.User) userResponse {
	return userResponse{
		ID:              u.ID,
		Email:           u.Email,
		Name:            u.Name,
		CurrentResumeID: u.CurrentResumeID,
	}
}
