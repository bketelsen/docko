package config

import (
	"crypto/rand"
	"encoding/base64"
	"log/slog"
	"os"
	"strconv"
)

type SiteConfig struct {
	Name           string
	URL            string
	DefaultOGImage string
}

type AuthConfig struct {
	AdminPassword string
	SessionSecret string
	SessionMaxAge int // hours
}

type Config struct {
	DatabaseURL string
	Port        string
	Env         string
	Site        SiteConfig
	Auth        AuthConfig
}

func Load() *Config {
	cfg := &Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		Port:        getEnvOrDefault("PORT", "3000"),
		Env:         getEnvOrDefault("ENV", "development"),
		Site: SiteConfig{
			Name:           getEnvOrDefault("SITE_NAME", "docko"),
			URL:            getEnvOrDefault("SITE_URL", "http://localhost:3000"),
			DefaultOGImage: getEnvOrDefault("DEFAULT_OG_IMAGE", "/static/images/og-default.png"),
		},
		Auth: AuthConfig{
			AdminPassword: os.Getenv("ADMIN_PASSWORD"),
			SessionSecret: getEnvOrDefault("SESSION_SECRET", generateDefaultSecret()),
			SessionMaxAge: getEnvIntOrDefault("SESSION_MAX_AGE", 24),
		},
	}

	if cfg.DatabaseURL == "" {
		slog.Error("DATABASE_URL environment variable is required")
		os.Exit(1)
	}

	if cfg.Auth.AdminPassword == "" {
		slog.Warn("ADMIN_PASSWORD not set - admin login will be disabled")
	}

	return cfg
}

func (c *Config) IsDevelopment() bool {
	return c.Env == "development"
}

func (c *Config) IsProduction() bool {
	return c.Env == "production"
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func generateDefaultSecret() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}
