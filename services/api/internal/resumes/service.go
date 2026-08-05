package resumes

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jobright/api/internal/auth"
	"github.com/jobright/api/internal/forge"
	"github.com/jobright/api/internal/middleware"
	"github.com/jobright/api/internal/models"
	"github.com/jobright/api/internal/resumeparse"
	"github.com/jobright/api/pkg/response"
	"gorm.io/gorm"
)

type Service struct {
	db        *gorm.DB
	auth      *auth.Service
	forge     *forge.Client
	uploadDir string
	maxBytes  int64
}

func NewService(db *gorm.DB, authSvc *auth.Service, forgeClient *forge.Client, uploadDir string, maxBytes int64) *Service {
	return &Service{db: db, auth: authSvc, forge: forgeClient, uploadDir: uploadDir, maxBytes: maxBytes}
}

type UploadResult struct {
	Resume  *models.Resume `json:"resume"`
	Profile gin.H          `json:"profile"`
}

func (s *Service) List(userID uuid.UUID) ([]models.Resume, error) {
	var items []models.Resume
	return items, s.db.Where("user_id = ?", userID).Order("created_at desc").Find(&items).Error
}

func (s *Service) Get(userID, id uuid.UUID) (*models.Resume, error) {
	var resume models.Resume
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&resume).Error; err != nil {
		return nil, err
	}
	return &resume, nil
}

func (s *Service) Upload(userID uuid.UUID, name, filename, contentType string, content []byte) (*UploadResult, error) {
	if int64(len(content)) == 0 {
		return nil, fmt.Errorf("empty file")
	}
	if int64(len(content)) > s.maxBytes {
		return nil, fmt.Errorf("file too large")
	}
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".pdf" && ext != ".docx" {
		return nil, fmt.Errorf("only pdf and docx are supported")
	}
	if err := os.MkdirAll(s.uploadDir, 0o755); err != nil {
		return nil, err
	}
	id := uuid.New()
	stored := filepath.Join(s.uploadDir, id.String()+ext)
	if err := os.WriteFile(stored, content, 0o644); err != nil {
		return nil, err
	}
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(filename), ext)
	}

	text, _ := resumeparse.ExtractText(filename, content)
	draft := resumeparse.ParseProfile(text)

	resume := &models.Resume{
		ID:          id,
		UserID:      userID,
		Name:        name,
		FilePath:    stored,
		FileName:    filepath.Base(filename),
		ContentType: contentType,
		ParsedText:  draft.RawText,
	}
	if err := s.db.Create(resume).Error; err != nil {
		_ = os.Remove(stored)
		return nil, err
	}

	user, err := s.auth.GetByID(userID)
	if err != nil {
		return nil, err
	}

	if draft.Name != "" {
		user.Name = draft.Name
	}
	if draft.Phone != "" {
		user.Phone = draft.Phone
	}
	if draft.LinkedIn != "" {
		user.LinkedIn = draft.LinkedIn
	}
	if draft.GitHub != "" {
		user.GitHub = draft.GitHub
	}
	if draft.Website != "" {
		user.Website = draft.Website
	}
	if draft.Location != "" {
		user.Location = draft.Location
	}
	if draft.Headline != "" {
		user.Headline = draft.Headline
	}
	if draft.CoverLetter != "" {
		user.CoverLetter = draft.CoverLetter
	}
	rid := resume.ID
	user.CurrentResumeID = &rid

	if err := s.db.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]any{
		"name":              user.Name,
		"phone":             user.Phone,
		"linkedin":          user.LinkedIn,
		"github":            user.GitHub,
		"website":           user.Website,
		"location":          user.Location,
		"headline":          user.Headline,
		"cover_letter":      user.CoverLetter,
		"current_resume_id": rid,
	}).Error; err != nil {
		return nil, fmt.Errorf("failed to update profile from resume: %w", err)
	}

	go s.syncForge(userID, resume, content)

	return &UploadResult{
		Resume: resume,
		Profile: gin.H{
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
		},
	}, nil
}

func (s *Service) Delete(userID, id uuid.UUID) error {
	resume, err := s.Get(userID, id)
	if err != nil {
		return err
	}
	if err := s.db.Delete(resume).Error; err != nil {
		return err
	}
	_ = os.Remove(resume.FilePath)
	_ = s.db.Model(&models.User{}).Where("id = ? AND current_resume_id = ?", userID, id).Update("current_resume_id", nil).Error
	return nil
}

func (s *Service) EnsureSynced(user *models.User, resume *models.Resume) (string, error) {
	if resume.ForgeResumeID != "" {
		return resume.ForgeResumeID, nil
	}
	content, err := os.ReadFile(resume.FilePath)
	if err != nil {
		return "", err
	}
	s.syncForge(user.ID, resume, content)
	if resume.ForgeResumeID == "" {
		return "", fmt.Errorf("failed to sync resume to resume_forge")
	}
	return resume.ForgeResumeID, nil
}

func (s *Service) syncForge(userID uuid.UUID, resume *models.Resume, content []byte) {
	if s.forge == nil || s.auth == nil {
		return
	}
	user, err := s.auth.GetByID(userID)
	if err != nil {
		return
	}
	token, err := s.auth.EnsureForgeToken(user)
	if err != nil {
		return
	}
	uploaded, err := s.forge.UploadResume(token, resume.Name, resume.FileName, content, resume.ContentType)
	if err != nil {
		return
	}
	_ = s.forge.ParseResume(token, uploaded.ID)
	resume.ForgeResumeID = uploaded.ID
	_ = s.db.Model(resume).Update("forge_resume_id", uploaded.ID).Error
}

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) List(c *gin.Context) {
	items, err := h.svc.List(middleware.UserID(c))
	if err != nil {
		response.Internal(c, "failed to list resumes")
		return
	}
	response.OK(c, items)
}

func (h *Handler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid resume id")
		return
	}
	resume, err := h.svc.Get(middleware.UserID(c), id)
	if err != nil {
		response.NotFound(c, "resume not found")
		return
	}
	response.OK(c, resume)
}

func (h *Handler) Upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "file is required")
		return
	}
	f, err := file.Open()
	if err != nil {
		response.BadRequest(c, "unable to read file")
		return
	}
	defer f.Close()
	content, err := io.ReadAll(io.LimitReader(f, h.svc.maxBytes+1))
	if err != nil {
		response.Internal(c, "failed to read upload")
		return
	}
	result, err := h.svc.Upload(middleware.UserID(c), c.PostForm("name"), file.Filename, file.Header.Get("Content-Type"), content)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Created(c, result)
}

func (h *Handler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid resume id")
		return
	}
	if err := h.svc.Delete(middleware.UserID(c), id); err != nil {
		response.NotFound(c, "resume not found")
		return
	}
	c.Status(204)
}

func (h *Handler) Download(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid resume id")
		return
	}
	resume, err := h.svc.Get(middleware.UserID(c), id)
	if err != nil {
		response.NotFound(c, "resume not found")
		return
	}
	c.FileAttachment(resume.FilePath, resume.FileName)
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.GET("", h.List)
	rg.GET("/:id", h.Get)
	rg.GET("/:id/file", h.Download)
	rg.POST("", h.Upload)
	rg.DELETE("/:id", h.Delete)
}
