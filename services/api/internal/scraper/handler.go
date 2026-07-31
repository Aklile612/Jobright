package scraper

import (
	"github.com/gin-gonic/gin"
	"github.com/jobright/api/pkg/response"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

type scrapeRequest struct {
	URLs []string `json:"urls" binding:"required,min=1"`
}

func (h *Handler) Scrape(c *gin.Context) {
	var req scrapeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "urls array is required")
		return
	}
	response.OK(c, h.svc.ScrapeURLs(req.URLs))
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.POST("/scrape", h.Scrape)
}
