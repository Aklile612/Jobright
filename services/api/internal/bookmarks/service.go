package bookmarks

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jobright/api/internal/middleware"
	"github.com/jobright/api/internal/models"
	"github.com/jobright/api/pkg/response"
	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) List(userID uuid.UUID) ([]models.Bookmark, error) {
	var items []models.Bookmark
	err := s.db.Preload("Job").Where("user_id = ?", userID).Order("created_at desc").Find(&items).Error
	return items, err
}

func (s *Service) Add(userID, jobID uuid.UUID) (*models.Bookmark, error) {
	var job models.Job
	if err := s.db.First(&job, "id = ?", jobID).Error; err != nil {
		return nil, err
	}
	var existing models.Bookmark
	err := s.db.Where("user_id = ? AND job_id = ?", userID, jobID).First(&existing).Error
	if err == nil {
		return &existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	bookmark := &models.Bookmark{UserID: userID, JobID: jobID}
	if err := s.db.Create(bookmark).Error; err != nil {
		return nil, err
	}
	bookmark.Job = &job
	return bookmark, nil
}

func (s *Service) Remove(userID, jobID uuid.UUID) error {
	return s.db.Where("user_id = ? AND job_id = ?", userID, jobID).Delete(&models.Bookmark{}).Error
}

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

type addRequest struct {
	JobID uuid.UUID `json:"job_id" binding:"required"`
}

func (h *Handler) List(c *gin.Context) {
	items, err := h.svc.List(middleware.UserID(c))
	if err != nil {
		response.Internal(c, "failed to list bookmarks")
		return
	}
	response.OK(c, items)
}

func (h *Handler) Add(c *gin.Context) {
	var req addRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid bookmark payload")
		return
	}
	item, err := h.svc.Add(middleware.UserID(c), req.JobID)
	if err != nil {
		response.BadRequest(c, "job not found")
		return
	}
	response.Created(c, item)
}

func (h *Handler) Remove(c *gin.Context) {
	jobID, err := uuid.Parse(c.Param("jobId"))
	if err != nil {
		response.BadRequest(c, "invalid job id")
		return
	}
	if err := h.svc.Remove(middleware.UserID(c), jobID); err != nil {
		response.Internal(c, "failed to remove bookmark")
		return
	}
	c.Status(204)
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.GET("", h.List)
	rg.POST("", h.Add)
	rg.DELETE("/:jobId", h.Remove)
}
