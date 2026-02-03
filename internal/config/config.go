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

type StorageConfig struct {
	Path string // Root path for document storage
}

type InboxConfig struct {
	DefaultPath    string // Default inbox path from env
	ErrorSubdir    string // Subdirectory for error files (default: "errors")
	MaxFileSizeMB  int    // Maximum file size in MB (default: 100)
	ScanIntervalMs int    // Interval between directory scans in ms (default: 1000)
}

type NetworkConfig struct {
	CredentialKey string // Key for encrypting network source credentials (required for network sources)
}

type Config struct {
	DatabaseURL string
	Port        string
	Env         string
	Site        SiteConfig
	Auth        AuthConfig
	Storage     StorageConfig
	Inbox       InboxConfig
	Network     NetworkConfig
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
		Storage: StorageConfig{
			Path: getEnvOrDefault("STORAGE_PATH", "./storage"),
		},
		Inbox: InboxConfig{
			DefaultPath:    os.Getenv("INBOX_PATH"), // Empty string if not set
			ErrorSubdir:    getEnvOrDefault("INBOX_ERROR_SUBDIR", "errors"),
			MaxFileSizeMB:  getEnvIntOrDefault("INBOX_MAX_FILE_SIZE_MB", 100),
			ScanIntervalMs: getEnvIntOrDefault("INBOX_SCAN_INTERVAL_MS", 1000),
		},
		Network: NetworkConfig{
			CredentialKey: os.Getenv("CREDENTIAL_ENCRYPTION_KEY"),
		},
	}

	if cfg.DatabaseURL == "" {
		slog.Error("DATABASE_URL environment variable is required")
		os.Exit(1)
	}

	if cfg.Auth.AdminPassword == "" {
		slog.Warn("ADMIN_PASSWORD not set - admin login will be disabled")
	}

	if cfg.Storage.Path == "./storage" {
		slog.Warn("STORAGE_PATH not set, using ./storage")
	}

	if cfg.Inbox.DefaultPath != "" {
		slog.Info("default inbox path configured", "path", cfg.Inbox.DefaultPath)
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
