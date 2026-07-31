package jobs

import "github.com/gin-gonic/gin"

func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.GET("", h.List)
	rg.GET("/:id", h.Get)
	rg.POST("", h.Create)
}
