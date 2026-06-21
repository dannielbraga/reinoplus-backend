package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	CORS     CORSConfig
}

type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type DatabaseConfig struct {
	URL string
}

type JWTConfig struct {
	PrivateKeyPath string
	PublicKeyPath  string
	AccessTTL      time.Duration
	RefreshTTL     time.Duration
}

type CORSConfig struct {
	AllowedOrigins []string
}

func Load() (Config, error) {
	_ = godotenv.Load()

	cfg := Config{
		Server: ServerConfig{
			Port:         getEnv("PORT", getEnv("SERVER_PORT", "8080")),
			ReadTimeout:  getDurationEnv("SERVER_READ_TIMEOUT", 10*time.Second),
			WriteTimeout: getDurationEnv("SERVER_WRITE_TIMEOUT", 15*time.Second),
		},
		Database: DatabaseConfig{
			URL: normalizeDatabaseURL(getEnv("DATABASE_URL", "postgres://reinoplus:reinoplus@localhost:5432/reinoplus?sslmode=disable")),
		},
		JWT: JWTConfig{
			PrivateKeyPath: getEnv("JWT_PRIVATE_KEY_PATH", "./keys/private.pem"),
			PublicKeyPath:  getEnv("JWT_PUBLIC_KEY_PATH", "./keys/public.pem"),
			AccessTTL:      getDurationEnv("JWT_ACCESS_TTL", 15*time.Minute),
			RefreshTTL:     getDurationEnv("JWT_REFRESH_TTL", 7*24*time.Hour),
		},
		CORS: CORSConfig{
			AllowedOrigins: getSliceEnv("CORS_ALLOWED_ORIGINS", []string{"*"}),
		},
	}

	if cfg.Database.URL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}

func normalizeDatabaseURL(raw string) string {
	if raw == "" || strings.Contains(raw, "sslmode=") {
		return raw
	}

	if strings.Contains(raw, "localhost") || strings.Contains(raw, "127.0.0.1") ||
		strings.Contains(raw, "@postgres:") || strings.Contains(raw, "railway.internal") {
		return raw
	}

	separator := "?"
	if strings.Contains(raw, "?") {
		separator = "&"
	}
	return raw + separator + "sslmode=require"
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getSliceEnv(key string, fallback []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parts := []string{}
	current := ""
	for _, ch := range value {
		if ch == ',' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
			continue
		}
		current += string(ch)
	}
	if current != "" {
		parts = append(parts, current)
	}
	if len(parts) == 0 {
		return fallback
	}
	return parts
}

func getIntEnv(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
