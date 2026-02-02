package testutil

import (
	"context"
	"os"
	"testing"

	"docko/internal/config"
	"docko/internal/database"
)

// NewTestDB creates a test database connection.
// Requires TEST_DATABASE_URL environment variable or uses DATABASE_URL.
func NewTestDB(t *testing.T) *database.DB {
	t.Helper()

	ctx := context.Background()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL or DATABASE_URL not set, skipping database test")
	}

	db, err := database.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

// NewTestConfig creates a test configuration.
func NewTestConfig(t *testing.T) *config.Config {
	t.Helper()

	return &config.Config{
		DatabaseURL: os.Getenv("TEST_DATABASE_URL"),
		Port:        "0",
		Env:         "test",
		Site: config.SiteConfig{
			Name: "docko",
			URL:  "http://localhost:3000",
		},
	}
}
