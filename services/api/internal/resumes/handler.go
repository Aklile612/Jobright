package resumes

import "github.com/gin-gonic/gin"

func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.GET("", h.List)
	rg.GET("/:id", h.Get)
	rg.GET("/:id/file", h.Download)
	rg.POST("", h.Upload)
	rg.DELETE("/:id", h.Delete)
}
