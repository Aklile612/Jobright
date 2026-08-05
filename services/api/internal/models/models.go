package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ApplicationStatus string

const (
	StatusSaved     ApplicationStatus = "saved"
	StatusApplied   ApplicationStatus = "applied"
	StatusInterview ApplicationStatus = "interview"
	StatusRejected  ApplicationStatus = "rejected"
	StatusOffer     ApplicationStatus = "offer"
)

type User struct {
	ID                uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	Email             string     `gorm:"uniqueIndex;size:255;not null" json:"email"`
	PasswordHash      string     `gorm:"not null" json:"-"`
	Name              string     `gorm:"size:120" json:"name"`
	Phone             string     `gorm:"size:40" json:"phone"`
	LinkedIn          string     `gorm:"column:linkedin;size:255" json:"linkedin"`
	GitHub            string     `gorm:"column:github;size:255" json:"github"`
	Website           string     `gorm:"size:255" json:"website"`
	Location          string     `gorm:"size:120" json:"location"`
	Headline          string     `gorm:"size:255" json:"headline"`
	Skills            string     `gorm:"type:text" json:"skills"`
	Education         string     `gorm:"type:text" json:"education"`
	CoverLetter       string     `gorm:"type:text" json:"cover_letter"`
	CurrentResumeID   *uuid.UUID `gorm:"type:uuid" json:"current_resume_id,omitempty"`
	ForgeAccessToken  string     `gorm:"type:text" json:"-"`
	ForgeRefreshToken string     `gorm:"type:text" json:"-"`
	ForgeUserID       string     `gorm:"size:64" json:"-"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

type Job struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Title       string    `gorm:"size:255;not null;index" json:"title"`
	Company     string    `gorm:"size:255;index" json:"company"`
	Description string    `gorm:"type:text;not null" json:"description"`
	Location    string    `gorm:"size:255" json:"location"`
	SourceURL   string    `gorm:"size:1024;uniqueIndex" json:"source_url"`
	SalaryRange string    `gorm:"size:120" json:"salary_range"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (j *Job) BeforeCreate(tx *gorm.DB) error {
	if j.ID == uuid.Nil {
		j.ID = uuid.New()
	}
	return nil
}

type Resume struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	UserID        uuid.UUID `gorm:"type:uuid;index;not null" json:"user_id"`
	Name          string    `gorm:"size:120;not null" json:"name"`
	FilePath      string    `gorm:"size:512;not null" json:"file_path"`
	FileName      string    `gorm:"size:255;not null" json:"file_name"`
	ContentType   string    `gorm:"size:120" json:"content_type"`
	ParsedText    string    `gorm:"type:text" json:"parsed_text,omitempty"`
	ForgeResumeID string    `gorm:"size:64" json:"forge_resume_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (r *Resume) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

type Application struct {
	ID              uuid.UUID         `gorm:"type:uuid;primaryKey" json:"id"`
	UserID          uuid.UUID         `gorm:"type:uuid;uniqueIndex:idx_user_job;not null" json:"user_id"`
	JobID           uuid.UUID         `gorm:"type:uuid;uniqueIndex:idx_user_job;not null" json:"job_id"`
	Status          ApplicationStatus `gorm:"size:32;not null;default:saved" json:"status"`
	MatchScore      *float64          `json:"match_score,omitempty"`
	MatchFeedback   []string          `gorm:"serializer:json" json:"match_feedback,omitempty"`
	MissingKeywords []string          `gorm:"serializer:json" json:"missing_keywords,omitempty"`
	ForgeJobID      string            `gorm:"size:64" json:"forge_job_id,omitempty"`
	ForgeReportID   string            `gorm:"size:64" json:"forge_report_id,omitempty"`
	ForgeVersionID  string            `gorm:"size:64" json:"forge_version_id,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	Job             *Job              `gorm:"foreignKey:JobID" json:"job,omitempty"`
}

func (a *Application) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	if a.Status == "" {
		a.Status = StatusSaved
	}
	return nil
}

type Bookmark struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_user_bookmark_job;not null" json:"user_id"`
	JobID     uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_user_bookmark_job;not null" json:"job_id"`
	CreatedAt time.Time `json:"created_at"`
	Job       *Job      `gorm:"foreignKey:JobID" json:"job,omitempty"`
}

func (b *Bookmark) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}
