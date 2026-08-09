package jobs

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jobright/api/internal/models"
	"github.com/jobright/api/pkg/response"
)

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
	limit := parseInt(c.Query("limit"), 24)
	offset := parseInt(c.Query("offset"), 0)
	jobs, total, err := h.svc.List(c.Query("q"), limit, offset)
	if err != nil {
		response.Internal(c, "failed to list jobs")
		return
	}
	response.OK(c, gin.H{
		"items":  jobs,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
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

func (h *Handler) Sync(c *gin.Context) {
	results, err := h.svc.SyncSoftwareJobs()
	if err != nil {
		response.Internal(c, "sync failed")
		return
	}
	response.OK(c, gin.H{"results": results})
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.GET("", h.List)
	rg.GET("/:id", h.Get)
	rg.POST("", h.Create)
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
