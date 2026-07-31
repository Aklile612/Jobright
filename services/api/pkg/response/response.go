package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func JSON(c *gin.Context, status int, data any) {
	c.JSON(status, data)
}

func OK(c *gin.Context, data any) {
	JSON(c, http.StatusOK, data)
}

func Created(c *gin.Context, data any) {
	JSON(c, http.StatusCreated, data)
}

func Error(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": message})
}

func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, message)
}

func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, message)
}

func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, message)
}

func Conflict(c *gin.Context, message string) {
	Error(c, http.StatusConflict, message)
}

func Internal(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, message)
}
