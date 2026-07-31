package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jobright/api/internal/applications"
	"github.com/jobright/api/internal/auth"
	"github.com/jobright/api/internal/bookmarks"
	"github.com/jobright/api/internal/config"
	"github.com/jobright/api/internal/extension"
	"github.com/jobright/api/internal/jobs"
	"github.com/jobright/api/internal/middleware"
	"github.com/jobright/api/internal/resumes"
	"github.com/jobright/api/internal/scraper"
	"github.com/jobright/api/internal/users"
)

type Deps struct {
	Config       config.Config
	Auth         *auth.Handler
	Users        *users.Handler
	Jobs         *jobs.Handler
	Resumes      *resumes.Handler
	Applications *applications.Handler
	Bookmarks    *bookmarks.Handler
	Extension    *extension.Handler
	Scraper      *scraper.Handler
}

func New(deps Deps) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger(), middleware.CORS(deps.Config.CORSOrigins))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api/v1")
	{
		authGroup := api.Group("/auth")
		authGroup.POST("/signup", deps.Auth.Register)
		authGroup.POST("/login", deps.Auth.Login)

		jobsPublic := api.Group("/jobs")
		jobsPublic.GET("", deps.Jobs.List)
		jobsPublic.GET("/:id", deps.Jobs.Get)

		protected := api.Group("")
		protected.Use(middleware.Auth(deps.Config.JWTSecret))
		protected.GET("/auth/me", deps.Auth.Me)
		deps.Users.Register(protected.Group("/users"))
		protected.POST("/jobs", deps.Jobs.Create)
		deps.Resumes.Register(protected.Group("/resumes"))
		deps.Applications.Register(protected.Group("/applications"))
		deps.Bookmarks.Register(protected.Group("/bookmarks"))
		deps.Scraper.Register(protected.Group("/admin"))
		deps.Extension.Register(protected.Group("/ext"))
	}

	return r
}
