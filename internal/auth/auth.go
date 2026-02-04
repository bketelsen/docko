package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/bketelsen/docko/internal/config"
	"github.com/bketelsen/docko/internal/database"
	"github.com/bketelsen/docko/internal/database/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

const (
	AdminUsername = "admin"
	TokenLength   = 32
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type Service struct {
	db  *database.DB
	cfg *config.Config
}

func NewService(db *database.DB, cfg *config.Config) *Service {
	return &Service{db: db, cfg: cfg}
}

// SyncAdminUser creates or updates the admin user based on ADMIN_PASSWORD env var
func (s *Service) SyncAdminUser(ctx context.Context) error {
	if s.cfg.Auth.AdminPassword == "" {
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(s.cfg.Auth.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user, err := s.db.Queries.GetAdminUserByUsername(ctx, AdminUsername)
	if errors.Is(err, pgx.ErrNoRows) {
		// User doesn't exist, create
		_, err = s.db.Queries.CreateAdminUser(ctx, sqlc.CreateAdminUserParams{
			Username:     AdminUsername,
			PasswordHash: string(hash),
		})
		if err != nil {
			return fmt.Errorf("failed to create admin user: %w", err)
		}
		slog.Info("created admin user")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get admin user: %w", err)
	}

	// User exists, check if password changed
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(s.cfg.Auth.AdminPassword)) != nil {
		// Password changed, update and invalidate sessions
		err = s.db.Queries.UpdateAdminUserPassword(ctx, sqlc.UpdateAdminUserPasswordParams{
			PasswordHash: string(hash),
			Username:     AdminUsername,
		})
		if err != nil {
			return fmt.Errorf("failed to update admin password: %w", err)
		}
		// Invalidate all existing sessions
		_ = s.db.Queries.DeleteAdminUserSessions(ctx, user.ID)
		slog.Info("updated admin password and invalidated existing sessions")
	}

	return nil
}

// ValidateCredentials checks username/password and returns user if valid
func (s *Service) ValidateCredentials(ctx context.Context, username, password string) (*sqlc.AdminUser, error) {
	user, err := s.db.Queries.GetAdminUserByUsername(ctx, username)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return nil, ErrInvalidCredentials
	}

	return &user, nil
}

// CreateSession creates a new session and returns the raw token (for cookie)
func (s *Service) CreateSession(ctx context.Context, userID uuid.UUID) (string, error) {
	// Generate random token
	tokenBytes := make([]byte, TokenLength)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	token := base64.URLEncoding.EncodeToString(tokenBytes)

	// Hash token for storage
	tokenHash := s.hashToken(token)

	expiresAt := time.Now().Add(time.Duration(s.cfg.Auth.SessionMaxAge) * time.Hour)

	_, err := s.db.Queries.CreateAdminSession(ctx, sqlc.CreateAdminSessionParams{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	return token, nil
}

// ValidateSession checks if token is valid and returns session data
func (s *Service) ValidateSession(ctx context.Context, token string) (*sqlc.GetAdminSessionByTokenHashRow, error) {
	tokenHash := s.hashToken(token)
	session, err := s.db.Queries.GetAdminSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("invalid session")
	}
	return &session, nil
}

// DeleteSession removes a session by token
func (s *Service) DeleteSession(ctx context.Context, token string) error {
	tokenHash := s.hashToken(token)
	return s.db.Queries.DeleteAdminSession(ctx, tokenHash)
}

func (s *Service) hashToken(token string) string {
	h := hmac.New(sha256.New, []byte(s.cfg.Auth.SessionSecret))
	h.Write([]byte(token))
	return base64.URLEncoding.EncodeToString(h.Sum(nil))
}

// CleanupExpiredSessions removes expired sessions (call periodically)
func (s *Service) CleanupExpiredSessions(ctx context.Context) error {
	return s.db.Queries.DeleteExpiredAdminSessions(ctx)
}
