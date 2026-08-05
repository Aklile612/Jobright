package jobs

import (
	"strings"

	"github.com/google/uuid"
	"github.com/jobright/api/internal/models"
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
