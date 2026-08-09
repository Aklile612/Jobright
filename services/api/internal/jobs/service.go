package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jobright/api/internal/cache"
	"github.com/jobright/api/internal/config"
	"github.com/jobright/api/internal/models"
	"gorm.io/gorm"
)

type Service struct {
	db     *gorm.DB
	cache  *cache.Store
	config config.Config
}

func NewService(db *gorm.DB, store *cache.Store, cfg config.Config) *Service {
	return &Service{db: db, cache: store, config: cfg}
}

const (
	listCacheTTL = 2 * time.Minute
	syncCooldown  = 15 * time.Minute
)

func (s *Service) List(q string, limit, offset int) ([]models.Job, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 24
	}
	if offset < 0 {
		offset = 0
	}
	q = strings.TrimSpace(q)
	ctx := context.Background()
	cacheKey := fmt.Sprintf("jobs:list:v2:%s:%d:%d", strings.ToLower(q), limit, offset)

	type pageCache struct {
		Items []models.Job `json:"items"`
		Total int64        `json:"total"`
	}

	if s.cache != nil {
		if raw, ok := s.cache.Get(ctx, cacheKey); ok {
			var cached pageCache
			if json.Unmarshal(raw, &cached) == nil {
				return cached.Items, cached.Total, nil
			}
		}
	}

	base := s.db.Model(&models.Job{})
	if q != "" {
		like := "%" + q + "%"
		base = base.Where("title ILIKE ? OR company ILIKE ? OR location ILIKE ? OR description ILIKE ?", like, like, like, like)
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Prefer curated boards first, then mix the rest (Adzuna/Muse/etc.) by recency.
	// Priority: JobRight → Remotive → RemoteOK → everything else.
	orderSQL := `
CASE
  WHEN source_url ILIKE '%jobright.ai%' THEN 0
  WHEN source_url ILIKE '%remotive.%' OR source_url ILIKE '%remotive.com%' THEN 1
  WHEN source_url ILIKE '%remoteok.%' OR source_url ILIKE '%remoteok.com%' THEN 2
  ELSE 3
END ASC,
created_at DESC`

	var jobs []models.Job
	if err := base.Order(orderSQL).Limit(limit).Offset(offset).Find(&jobs).Error; err != nil {
		return nil, 0, err
	}

	if s.cache != nil {
		if raw, err := json.Marshal(pageCache{Items: jobs, Total: total}); err == nil {
			s.cache.Set(ctx, cacheKey, raw, listCacheTTL)
		}
	}
	return jobs, total, nil
}

func (s *Service) Get(id uuid.UUID) (*models.Job, error) {
	var job models.Job
	if err := s.db.First(&job, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

func (s *Service) Create(job *models.Job) error {
	if err := s.db.Create(job).Error; err != nil {
		return err
	}
	s.invalidateJobCaches()
	return nil
}

func (s *Service) UpsertBySourceURL(job *models.Job) error {
	if job.SourceURL == "" {
		err := s.db.Create(job).Error
		if err == nil {
			s.invalidateJobCaches()
		}
		return err
	}
	var existing models.Job
	err := s.db.Where("source_url = ?", job.SourceURL).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		err = s.db.Create(job).Error
		if err == nil {
			s.invalidateJobCaches()
		}
		return err
	}
	if err != nil {
		return err
	}
	existing.Title = job.Title
	existing.Company = job.Company
	existing.Description = job.Description
	existing.Location = job.Location
	existing.SalaryRange = job.SalaryRange
	if err := s.db.Save(&existing).Error; err != nil {
		return err
	}
	s.invalidateJobCaches()
	return nil
}

func (s *Service) invalidateJobCaches() {
	if s.cache == nil {
		return
	}
	s.cache.DelByPrefix(context.Background(), "jobs:list:")
}

// BeginSync returns false when a sync ran recently (cooldown).
func (s *Service) BeginSync() bool {
	if s.cache == nil {
		return true
	}
	ctx := context.Background()
	if _, ok := s.cache.Get(ctx, "jobs:sync:cooldown"); ok {
		return false
	}
	return true
}

func (s *Service) EndSync() {
	if s.cache == nil {
		return
	}
	s.cache.Set(context.Background(), "jobs:sync:cooldown", []byte("1"), syncCooldown)
	s.invalidateJobCaches()
}
