package jobs

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

func (s *Service) List(q string, limit, offset int) ([]models.Job, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	tx := s.db.Model(&models.Job{}).Order("created_at desc").Limit(limit).Offset(offset)
	q = strings.TrimSpace(q)
	if q != "" {
		like := "%" + q + "%"
		tx = tx.Where("title ILIKE ? OR company ILIKE ? OR location ILIKE ?", like, like, like)
	}
	var jobs []models.Job
	return jobs, tx.Find(&jobs).Error
}

func (s *Service) Get(id uuid.UUID) (*models.Job, error) {
	var job models.Job
	if err := s.db.First(&job, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

func (s *Service) Create(job *models.Job) error {
	return s.db.Create(job).Error
}

func (s *Service) UpsertBySourceURL(job *models.Job) error {
	if job.SourceURL == "" {
		return s.db.Create(job).Error
	}
	var existing models.Job
	err := s.db.Where("source_url = ?", job.SourceURL).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return s.db.Create(job).Error
	}
	if err != nil {
		return err
	}
	existing.Title = job.Title
	existing.Company = job.Company
	existing.Description = job.Description
	existing.Location = job.Location
	existing.SalaryRange = job.SalaryRange
	return s.db.Save(&existing).Error
}

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

type createJobRequest struct {
	Title       string `json:"title" binding:"required"`
	Company     string `json:"company"`
	Description string `json:"description" binding:"required"`
	Location    string `json:"location"`
	SourceURL   string `json:"source_url"`
	SalaryRange string `json:"salary_range"`
}

func (h *Handler) List(c *gin.Context) {
	jobs, err := h.svc.List(c.Query("q"), parseInt(c.Query("limit"), 20), parseInt(c.Query("offset"), 0))
	if err != nil {
		response.Internal(c, "failed to list jobs")
		return
	}
	response.OK(c, jobs)
}

func (h *Handler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid job id")
		return
	}
	job, err := h.svc.Get(id)
	if err != nil {
		response.NotFound(c, "job not found")
		return
	}
	response.OK(c, job)
}

func (h *Handler) Create(c *gin.Context) {
	var req createJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid job payload")
		return
	}
	job := &models.Job{
		Title:       strings.TrimSpace(req.Title),
		Company:     strings.TrimSpace(req.Company),
		Description: strings.TrimSpace(req.Description),
		Location:    strings.TrimSpace(req.Location),
		SourceURL:   strings.TrimSpace(req.SourceURL),
		SalaryRange: strings.TrimSpace(req.SalaryRange),
	}
	if err := h.svc.Create(job); err != nil {
		response.Internal(c, "failed to create job")
		return
	}
	response.Created(c, job)
}

func parseInt(v string, fallback int) int {
	if v == "" {
		return fallback
	}
	var n int
	for _, ch := range v {
		if ch < '0' || ch > '9' {
			return fallback
		}
		n = n*10 + int(ch-'0')
	}
	return n
}
