package extension

import (
	"github.com/gin-gonic/gin"
	"github.com/jobright/api/internal/auth"
	"github.com/jobright/api/internal/middleware"
	"github.com/jobright/api/internal/models"
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

type autofillResponse struct {
	Name         string `json:"name"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	LinkedIn     string `json:"linkedin"`
	GitHub       string `json:"github"`
	Website      string `json:"website"`
	Location     string `json:"location"`
	Headline     string `json:"headline"`
	CoverLetter  string `json:"cover_letter"`
	ResumeID     string `json:"resume_id,omitempty"`
	ResumeName   string `json:"resume_name,omitempty"`
	ResumeFile   string `json:"resume_file_name,omitempty"`
	ContentType  string `json:"content_type,omitempty"`
	HasResume    bool   `json:"has_resume"`
	DownloadPath string `json:"download_path,omitempty"`
}

func (h *Handler) AutofillData(c *gin.Context) {
	user, err := h.auth.GetByID(middleware.UserID(c))
	if err != nil {
		response.NotFound(c, "user not found")
		return
	}
	out := autofillResponse{
		Name:        user.Name,
		Email:       user.Email,
		Phone:       user.Phone,
		LinkedIn:    user.LinkedIn,
		GitHub:      user.GitHub,
		Website:     user.Website,
		Location:    user.Location,
		Headline:    user.Headline,
		CoverLetter: user.CoverLetter,
	}
	if user.CurrentResumeID != nil {
		var resume models.Resume
		if err := h.db.First(&resume, "id = ? AND user_id = ?", *user.CurrentResumeID, user.ID).Error; err == nil {
			out.HasResume = true
			out.ResumeID = resume.ID.String()
			out.ResumeName = resume.Name
			out.ResumeFile = resume.FileName
			out.ContentType = resume.ContentType
			out.DownloadPath = "/api/v1/resumes/" + resume.ID.String() + "/file"
		}
	}
	response.OK(c, out)
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.GET("/autofill-data", h.AutofillData)
}
