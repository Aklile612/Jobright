package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port            string
	DatabaseURL     string
	JWTSecret       string
	JWTExpiry       time.Duration
	CORSOrigins     []string
	ResumeForgeURL  string
	UploadDir       string
	MaxUploadBytes  int64
	GeminiAPIKey    string
	GeminiModel     string
}

func Load() Config {
	return Config{
		Port:           getenv("API_PORT", "8080"),
		DatabaseURL:    getenv("DATABASE_URL", "postgres://jobright:jobright@localhost:5432/jobright?sslmode=disable"),
		JWTSecret:      getenv("JWT_SECRET", "dev-secret-change-me"),
		JWTExpiry:      durationHours("JWT_EXPIRY_HOURS", 72),
		CORSOrigins:    splitCSV(getenv("CORS_ORIGINS", "http://localhost:3000")),
		ResumeForgeURL: strings.TrimRight(getenv("RESUME_FORGE_URL", "http://localhost:8000"), "/"),
		UploadDir:      getenv("UPLOAD_DIR", "../../storage/resumes"),
		MaxUploadBytes: int64(getenvInt("MAX_UPLOAD_BYTES", 6*1024*1024)),
		GeminiAPIKey:   getenv("GEMINI_API_KEY", ""),
		GeminiModel:    getenv("GEMINI_MODEL", "gemini-2.0-flash"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func durationHours(key string, fallback int) time.Duration {
	return time.Duration(getenvInt(key, fallback)) * time.Hour
}

func splitCSV(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
