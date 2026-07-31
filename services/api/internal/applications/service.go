package applications

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jobright/api/internal/auth"
	"github.com/jobright/api/internal/forge"
	"github.com/jobright/api/internal/middleware"
	"github.com/jobright/api/internal/models"
	"github.com/jobright/api/internal/resumes"
	"github.com/jobright/api/pkg/response"
	"gorm.io/gorm"
)

type Service struct {
	db      *gorm.DB
	auth    *auth.Service
	resumes *resumes.Service
	forge   *forge.Client
}

func NewService(db *gorm.DB, authSvc *auth.Service, resumeSvc *resumes.Service, forgeClient *forge.Client) *Service {
	return &Service{db: db, auth: authSvc, resumes: resumeSvc, forge: forgeClient}
}

func (s *Service) List(userID uuid.UUID) ([]models.Application, error) {
	var items []models.Application
	err := s.db.Preload("Job").Where("user_id = ?", userID).Order("updated_at desc").Find(&items).Error
	return items, err
}

func (s *Service) Get(userID, id uuid.UUID) (*models.Application, error) {
	var app models.Application
	if err := s.db.Preload("Job").Where("id = ? AND user_id = ?", id, userID).First(&app).Error; err != nil {
		return nil, err
	}
	return &app, nil
}

func (s *Service) Upsert(userID, jobID uuid.UUID, status models.ApplicationStatus) (*models.Application, error) {
	var job models.Job
	if err := s.db.First(&job, "id = ?", jobID).Error; err != nil {
		return nil, err
	}
	var app models.Application
	err := s.db.Where("user_id = ? AND job_id = ?", userID, jobID).First(&app).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		app = models.Application{UserID: userID, JobID: jobID, Status: status}
		if status == "" {
			app.Status = models.StatusSaved
		}
		if err := s.db.Create(&app).Error; err != nil {
			return nil, err
		}
		return s.Get(userID, app.ID)
	}
	if err != nil {
		return nil, err
	}
	if status != "" {
		app.Status = status
		if err := s.db.Save(&app).Error; err != nil {
			return nil, err
		}
	}
	return s.Get(userID, app.ID)
}

func (s *Service) UpdateStatus(userID, id uuid.UUID, status models.ApplicationStatus) (*models.Application, error) {
	app, err := s.Get(userID, id)
	if err != nil {
		return nil, err
	}
	app.Status = status
	if err := s.db.Save(app).Error; err != nil {
		return nil, err
	}
	return app, nil
}

func (s *Service) Score(userID, applicationID uuid.UUID) (*models.Application, error) {
	app, user, resume, job, token, err := s.prepareForge(userID, applicationID)
	if err != nil {
		return nil, err
	}
	forgeResumeID, err := s.resumes.EnsureSynced(user, resume)
	if err != nil {
		return nil, err
	}
	forgeJobID, err := s.ensureForgeJob(token, app, job)
	if err != nil {
		return nil, err
	}
	report, err := s.forge.Analyze(token, forgeResumeID, forgeJobID)
	if err != nil {
		return nil, err
	}
	score := report.MatchScore
	app.MatchScore = &score
	app.MatchFeedback = append(report.Suggestions, append(report.Strengths, report.Weaknesses...)...)
	app.MissingKeywords = report.MissingKeywords
	app.ForgeReportID = report.ID
	if err := s.db.Save(app).Error; err != nil {
		return nil, err
	}
	return s.Get(userID, app.ID)
}

func (s *Service) ForgeResume(userID, applicationID uuid.UUID) (*models.Application, *forge.OptimizeResult, error) {
	app, user, resume, job, token, err := s.prepareForge(userID, applicationID)
	if err != nil {
		return nil, nil, err
	}
	forgeResumeID, err := s.resumes.EnsureSynced(user, resume)
	if err != nil {
		return nil, nil, err
	}
	forgeJobID, err := s.ensureForgeJob(token, app, job)
	if err != nil {
		return nil, nil, err
	}
	result, err := s.forge.Optimize(token, forgeResumeID, forgeJobID)
	if err != nil {
		return nil, nil, err
	}
	score := result.FinalATS.MatchScore
	app.MatchScore = &score
	app.MatchFeedback = append(result.FinalATS.Suggestions, append(result.FinalATS.Strengths, result.FinalATS.Weaknesses...)...)
	app.MissingKeywords = result.FinalATS.MissingKeywords
	app.ForgeReportID = result.FinalATS.ID
	app.ForgeVersionID = result.Version.ID
	if err := s.db.Save(app).Error; err != nil {
		return nil, nil, err
	}
	updated, err := s.Get(userID, app.ID)
	return updated, result, err
}

func (s *Service) prepareForge(userID, applicationID uuid.UUID) (*models.Application, *models.User, *models.Resume, *models.Job, string, error) {
	app, err := s.Get(userID, applicationID)
	if err != nil {
		return nil, nil, nil, nil, "", err
	}
	user, err := s.auth.GetByID(userID)
	if err != nil {
		return nil, nil, nil, nil, "", err
	}
	token, err := s.auth.EnsureForgeToken(user)
	if err != nil {
		return nil, nil, nil, nil, "", err
	}
	if user.CurrentResumeID == nil {
		return nil, nil, nil, nil, "", fmt.Errorf("upload a resume before scoring")
	}
	resume, err := s.resumes.Get(userID, *user.CurrentResumeID)
	if err != nil {
		return nil, nil, nil, nil, "", fmt.Errorf("current resume not found")
	}
	if app.Job == nil {
		return nil, nil, nil, nil, "", fmt.Errorf("job not found")
	}
	return app, user, resume, app.Job, token, nil
}

func (s *Service) ensureForgeJob(token string, app *models.Application, job *models.Job) (string, error) {
	if app.ForgeJobID != "" {
		return app.ForgeJobID, nil
	}
	jobURL := ""
	if strings.HasPrefix(job.SourceURL, "http://") || strings.HasPrefix(job.SourceURL, "https://") {
		jobURL = job.SourceURL
	}
	created, err := s.forge.CreateJob(token, job.Title, job.Company, job.Description, jobURL)
	if err != nil {
		return "", err
	}
	if err := s.forge.ParseJob(token, created.ID); err != nil {
		return "", err
	}
	app.ForgeJobID = created.ID
	if err := s.db.Model(app).Update("forge_job_id", created.ID).Error; err != nil {
		return "", err
	}
	return created.ID, nil
}

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

type upsertRequest struct {
	JobID  uuid.UUID               `json:"job_id" binding:"required"`
	Status models.ApplicationStatus `json:"status"`
}

type statusRequest struct {
	Status models.ApplicationStatus `json:"status" binding:"required"`
}

func (h *Handler) List(c *gin.Context) {
	items, err := h.svc.List(middleware.UserID(c))
	if err != nil {
		response.Internal(c, "failed to list applications")
		return
	}
	response.OK(c, items)
}

func (h *Handler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid application id")
		return
	}
	app, err := h.svc.Get(middleware.UserID(c), id)
	if err != nil {
		response.NotFound(c, "application not found")
		return
	}
	response.OK(c, app)
}

func (h *Handler) Upsert(c *gin.Context) {
	var req upsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid application payload")
		return
	}
	app, err := h.svc.Upsert(middleware.UserID(c), req.JobID, req.Status)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Created(c, app)
}

func (h *Handler) UpdateStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid application id")
		return
	}
	var req statusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid status payload")
		return
	}
	app, err := h.svc.UpdateStatus(middleware.UserID(c), id, req.Status)
	if err != nil {
		response.NotFound(c, "application not found")
		return
	}
	response.OK(c, app)
}

func (h *Handler) Score(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid application id")
		return
	}
	app, err := h.svc.Score(middleware.UserID(c), id)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, app)
}

func (h *Handler) Forge(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid application id")
		return
	}
	app, result, err := h.svc.ForgeResume(middleware.UserID(c), id)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, gin.H{
		"application": app,
		"optimization": result,
	})
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.GET("", h.List)
	rg.GET("/:id", h.Get)
	rg.POST("", h.Upsert)
	rg.PATCH("/:id/status", h.UpdateStatus)
	rg.POST("/:id/score", h.Score)
	rg.POST("/:id/forge", h.Forge)
}
