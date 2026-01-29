package config

import (
	"log"
	"os"
)

type Config struct {
	Port               string
	DBPath             string
	GoogleClientID     string
	GoogleClientSecret string
	JWTSecret          string
	FrontendDistPath   string
	BaseURL            string
}

func Load() *Config {
	cfg := &Config{
		Port:               getEnv("PORT", "8080"),
		DBPath:             getEnv("DB_PATH", "data/bets.db"),
		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		JWTSecret:          getEnv("JWT_SECRET", "change-me-in-production"),
		FrontendDistPath:   getEnv("FRONTEND_DIST_PATH", "frontend/dist"),
		BaseURL:            getEnv("BASE_URL", "http://localhost:8080"),
	}

	if cfg.GoogleClientID == "" || cfg.GoogleClientSecret == "" {
		log.Println("WARNING: Google OAuth credentials not set. Auth will not work.")
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
