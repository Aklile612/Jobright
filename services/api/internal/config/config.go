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
	GroqAPIKey      string
	GroqModel       string
	RedisURL        string
	AdzunaAppID     string
	AdzunaAppKey    string
	AdzunaCountries []string
	MuseAPIKey      string
	JSearchAPIKey   string
	AIRateLimit     int
	AIRateWindowSec int
}

func Load() Config {
	port := getenv("API_PORT", "")
	if port == "" {
		port = getenv("PORT", "8080") // EthioDeploy / Render / Railway
	}
	corsRaw := getenv("CORS_ORIGINS", "*")
	origins := splitCSV(corsRaw)
	// Unless CORS_STRICT is on, always allow any browser origin. Operators often set a
	// mistyped CORS_ORIGINS on EthioDeploy; that previously broke Vercel signup entirely.
	strict := strings.EqualFold(getenv("CORS_STRICT", ""), "true") || strings.EqualFold(getenv("CORS_STRICT", ""), "1")
	if !strict {
		hasStar := false
		for _, o := range origins {
			if o == "*" {
				hasStar = true
				break
			}
		}
		if !hasStar {
			origins = append(origins, "*")
		}
	}
	return Config{
		Port:            port,
		DatabaseURL:     getenv("DATABASE_URL", "postgres://jobright:jobright@localhost:5432/jobright?sslmode=disable"),
		JWTSecret:       getenv("JWT_SECRET", "dev-secret-change-me"),
		JWTExpiry:       durationHours("JWT_EXPIRY_HOURS", 72),
		CORSOrigins:     origins,
		ResumeForgeURL:  strings.TrimRight(getenv("RESUME_FORGE_URL", "http://localhost:8000"), "/"),
		UploadDir:       getenv("UPLOAD_DIR", "../../storage/resumes"),
		MaxUploadBytes:  int64(getenvInt("MAX_UPLOAD_BYTES", 6*1024*1024)),
		GeminiAPIKey:    getenv("GEMINI_API_KEY", ""),
		GeminiModel:     getenv("GEMINI_MODEL", "gemini-2.0-flash"),
		GroqAPIKey:      getenv("GROQ_API_KEY", ""),
		GroqModel:       getenv("GROQ_MODEL", "llama-3.1-8b-instant"),
		RedisURL:        getenv("REDIS_URL", ""),
		AdzunaAppID:     getenv("ADZUNA_APP_ID", ""),
		AdzunaAppKey:    getenv("ADZUNA_APP_KEY", ""),
		AdzunaCountries: splitCSV(getenv("ADZUNA_COUNTRIES", "us,gb,de,ca")),
		MuseAPIKey:      getenv("MUSE_API_KEY", ""),
		JSearchAPIKey:   firstNonEmpty(getenv("JSEARCH_API_KEY", ""), getenv("RAPIDAPI_KEY", "")),
		AIRateLimit:     getenvInt("AI_RATE_LIMIT", 20),
		AIRateWindowSec: getenvInt("AI_RATE_WINDOW_SEC", 60),
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

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func splitCSV(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.Trim(p, `"'`)
		p = strings.TrimRight(p, "/")
		if p != "" {
			out = append(out, p)
		}
	}
	// Also accept FRONTEND_URL / WEB_URL if operators set those instead of CORS_ORIGINS.
	for _, key := range []string{"FRONTEND_URL", "WEB_URL", "PUBLIC_WEB_URL"} {
		if extra := strings.TrimSpace(os.Getenv(key)); extra != "" {
			extra = strings.Trim(extra, `"'`)
			extra = strings.TrimRight(extra, "/")
			out = append(out, extra)
		}
	}
	return out
}
