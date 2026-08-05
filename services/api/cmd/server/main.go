package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/jobright/api/internal/ai"
	"github.com/jobright/api/internal/applications"
	"github.com/jobright/api/internal/auth"
	"github.com/jobright/api/internal/bookmarks"
	"github.com/jobright/api/internal/config"
	"github.com/jobright/api/internal/database"
	"github.com/jobright/api/internal/extension"
	"github.com/jobright/api/internal/forge"
	"github.com/jobright/api/internal/gemini"
	"github.com/jobright/api/internal/jobs"
	"github.com/jobright/api/internal/resumes"
	"github.com/jobright/api/internal/router"
	"github.com/jobright/api/internal/scraper"
	"github.com/jobright/api/internal/users"
)

func main() {
	_ = godotenv.Load()
	_ = godotenv.Load(filepath.Join("..", "..", "..", ".env"))
	_ = godotenv.Load(filepath.Join("..", "..", ".env"))

	cfg := config.Load()
	if err := os.MkdirAll(cfg.UploadDir, 0o755); err != nil {
		log.Fatalf("upload dir: %v", err)
	}

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	forgeClient := forge.NewClient(cfg.ResumeForgeURL)
	geminiClient := gemini.NewClient(cfg.GeminiAPIKey, cfg.GeminiModel)
	authSvc := auth.NewService(db, forgeClient, cfg.JWTSecret, cfg.JWTExpiry)
	jobSvc := jobs.NewService(db)
	resumeSvc := resumes.NewService(db, authSvc, forgeClient, cfg.UploadDir, cfg.MaxUploadBytes)
	appSvc := applications.NewService(db, authSvc, resumeSvc, forgeClient)
	bookmarkSvc := bookmarks.NewService(db)
	scraperSvc := scraper.NewService(jobSvc)

	engine := router.New(router.Deps{
		Config:       cfg,
		Auth:         auth.NewHandler(authSvc),
		Users:        users.NewHandler(db, authSvc),
		Jobs:         jobs.NewHandler(jobSvc),
		Resumes:      resumes.NewHandler(resumeSvc),
		Applications: applications.NewHandler(appSvc),
		Bookmarks:    bookmarks.NewHandler(bookmarkSvc),
		Extension:    extension.NewHandler(db, authSvc),
		Scraper:      scraper.NewHandler(scraperSvc),
		AI:           ai.NewHandler(db, authSvc, geminiClient),
	})

	geminiStatus := "off"
	if geminiClient.Enabled() {
		geminiStatus = cfg.GeminiModel
	}
	log.Printf("api listening on :%s (resume_forge=%s gemini=%s)", cfg.Port, cfg.ResumeForgeURL, geminiStatus)
	if err := engine.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
