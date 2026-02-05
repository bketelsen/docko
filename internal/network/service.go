package network

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/bketelsen/docko/internal/config"
	"github.com/bketelsen/docko/internal/database"
	"github.com/bketelsen/docko/internal/database/sqlc"
	"github.com/bketelsen/docko/internal/document"
)

const (
	// MaxConsecutiveFailures before auto-disabling a source
	MaxConsecutiveFailures = 5
	// DefaultBatchSize for sync operations
	DefaultBatchSize = 100
	// TempFilePrefix for downloaded files
	TempFilePrefix = "network-sync-"
)

// Event action constants
const (
	ActionImported  = "imported"
	ActionDuplicate = "duplicate"
	ActionError     = "error"
	ActionSkipped   = "skipped"
)

// Service coordinates network source sync operations.
type Service struct {
	db     *database.DB
	docSvc *document.Service
	cfg    *config.Config
	crypto *CredentialCrypto
	poller *Poller
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// New creates a new network Service.
func New(db *database.DB, docSvc *document.Service, cfg *config.Config) *Service {
	credentialKey := cfg.Network.CredentialKey
	if credentialKey == "" {
		slog.Warn("CREDENTIAL_ENCRYPTION_KEY not set - network source credentials will not be secure")
		// Use a fallback for development, but warn loudly
		credentialKey = "insecure-dev-key-do-not-use-in-production"
	}

	return &Service{
		db:     db,
		docSvc: docSvc,
		cfg:    cfg,
		crypto: NewCredentialCrypto(credentialKey),
	}
}

// Start initializes and starts the background poller.
func (s *Service) Start(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)

	// Create and start poller
	s.poller = NewPoller(s, 5*time.Minute)
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.poller.Run(s.ctx); err != nil && err != context.Canceled {
			slog.Error("poller error", "error", err)
		}
	}()

	slog.Info("network service started")
	return nil
}

// Stop gracefully shuts down the network service.
func (s *Service) Stop() error {
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
	slog.Info("network service stopped")
	return nil
}

// TestConnection tests connectivity to a network source.
func (s *Service) TestConnection(ctx context.Context, cfg *sqlc.NetworkSource) error {
	source, err := NewSourceFromConfig(cfg, s.crypto)
	if err != nil {
		return fmt.Errorf("create source: %w", err)
	}
	defer func() { _ = source.Close() }()

	return source.Test(ctx)
}

// SyncSource synchronizes a single network source.
// Returns number of files imported and any error.
func (s *Service) SyncSource(ctx context.Context, sourceID uuid.UUID) (int, error) {
	// Load source config
	cfg, err := s.db.Queries.GetNetworkSource(ctx, sourceID)
	if err != nil {
		return 0, fmt.Errorf("get source: %w", err)
	}

	if !cfg.Enabled {
		return 0, fmt.Errorf("source is disabled")
	}

	// Create network source client
	source, err := NewSourceFromConfig(&cfg, s.crypto)
	if err != nil {
		s.recordSyncFailure(ctx, &cfg, err)
		return 0, fmt.Errorf("create source: %w", err)
	}
	defer func() { _ = source.Close() }()

	slog.Info("starting sync", "source", cfg.Name, "host", cfg.Host)

	// List PDF files
	files, err := source.ListPDFs(ctx)
	if err != nil {
		s.recordSyncFailure(ctx, &cfg, err)
		return 0, fmt.Errorf("list files: %w", err)
	}

	slog.Info("found PDF files", "source", cfg.Name, "count", len(files))

	// Apply batch size limit
	if len(files) > int(cfg.BatchSize) {
		files = files[:cfg.BatchSize]
		slog.Info("applying batch limit", "source", cfg.Name, "batch_size", cfg.BatchSize)
	}

	// Process each file
	imported := 0
	for _, file := range files {
		select {
		case <-ctx.Done():
			return imported, ctx.Err()
		default:
		}

		if err := s.importFile(ctx, source, &cfg, file); err != nil {
			slog.Warn("failed to import file", "file", file.Path, "error", err)
			s.logEvent(ctx, cfg.ID, file.Name, file.Path, ActionError, nil, err.Error())
			continue
		}
		imported++
	}

	// Reset failure count on successful sync
	if err := s.db.Queries.ResetConsecutiveFailures(ctx, cfg.ID); err != nil {
		slog.Warn("failed to reset failure count", "error", err)
	}

	// Update last sync time
	s.updateSyncStatus(ctx, cfg.ID, nil)

	slog.Info("sync complete", "source", cfg.Name, "imported", imported, "total", len(files))
	return imported, nil
}

// SyncAll synchronizes all enabled sources.
func (s *Service) SyncAll(ctx context.Context) error {
	sources, err := s.db.Queries.ListEnabledNetworkSources(ctx)
	if err != nil {
		return fmt.Errorf("list sources: %w", err)
	}

	for _, source := range sources {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if _, err := s.SyncSource(ctx, source.ID); err != nil {
			slog.Warn("sync failed", "source", source.Name, "error", err)
			// Continue with other sources
		}
	}
	return nil
}

// importFile downloads and ingests a single file.
func (s *Service) importFile(ctx context.Context, source NetworkSource, cfg *sqlc.NetworkSource, file RemoteFile) error {
	// Create temp file for download
	tmpFile, err := os.CreateTemp("", TempFilePrefix+"*.pdf")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	// Download file
	if err := source.ReadFile(ctx, file.Path, tmpFile); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("download: %w", err)
	}
	_ = tmpFile.Close()

	// Ingest through document service
	doc, isDupe, err := s.docSvc.Ingest(ctx, tmpPath, file.Name)
	if err != nil {
		return fmt.Errorf("ingest: %w", err)
	}

	// Handle result
	if isDupe {
		s.logEvent(ctx, cfg.ID, file.Name, file.Path, ActionDuplicate, &doc.ID, "")
		slog.Debug("duplicate file", "file", file.Name, "existing_doc", doc.ID)
	} else {
		s.logEvent(ctx, cfg.ID, file.Name, file.Path, ActionImported, &doc.ID, "")
		if err := s.db.Queries.IncrementFilesImported(ctx, cfg.ID); err != nil {
			slog.Warn("failed to increment import count", "error", err)
		}
		slog.Info("imported file", "file", file.Name, "doc_id", doc.ID)
	}

	// Handle post-import action
	if err := s.handlePostImportAction(ctx, source, cfg, file); err != nil {
		slog.Warn("post-import action failed", "file", file.Path, "error", err)
		// Don't fail the import for post-action errors
	}

	return nil
}

// handlePostImportAction processes file after successful import.
func (s *Service) handlePostImportAction(ctx context.Context, source NetworkSource, cfg *sqlc.NetworkSource, file RemoteFile) error {
	switch cfg.PostImportAction {
	case sqlc.PostImportActionLeave:
		// Do nothing
		return nil

	case sqlc.PostImportActionDelete:
		return source.DeleteFile(ctx, file.Path)

	case sqlc.PostImportActionMove:
		// Move to subfolder
		subfolder := "imported"
		if cfg.MoveSubfolder != nil && *cfg.MoveSubfolder != "" {
			subfolder = *cfg.MoveSubfolder
		}

		// Construct destination path
		dir := filepath.Dir(file.Path)
		destPath := filepath.Join(dir, subfolder, file.Name)
		return source.MoveFile(ctx, file.Path, destPath)

	default:
		return nil
	}
}

// recordSyncFailure increments failure count and potentially disables source.
func (s *Service) recordSyncFailure(ctx context.Context, cfg *sqlc.NetworkSource, syncErr error) {
	errMsg := syncErr.Error()
	s.updateSyncStatus(ctx, cfg.ID, &errMsg)

	// Increment failure count
	newCount, err := s.db.Queries.IncrementConsecutiveFailures(ctx, cfg.ID)
	if err != nil {
		slog.Warn("failed to increment failure count", "error", err)
		return
	}

	// Auto-disable after too many failures
	if newCount >= MaxConsecutiveFailures {
		if err := s.db.Queries.DisableNetworkSource(ctx, cfg.ID); err != nil {
			slog.Warn("failed to disable source", "error", err)
			return
		}
		slog.Warn("source auto-disabled after consecutive failures",
			"source", cfg.Name,
			"failures", newCount,
		)
	}
}

// updateSyncStatus updates the source's sync timestamp and error.
func (s *Service) updateSyncStatus(ctx context.Context, sourceID uuid.UUID, errMsg *string) {
	state := "connected"
	if errMsg != nil {
		state = "error"
	}

	// Get current failure count
	source, _ := s.db.Queries.GetNetworkSource(ctx, sourceID)

	err := s.db.Queries.UpdateNetworkSourceStatus(ctx, sqlc.UpdateNetworkSourceStatusParams{
		ID:                  sourceID,
		ConnectionState:     &state,
		ConsecutiveFailures: source.ConsecutiveFailures,
		LastSyncAt:          pgtype.Timestamptz{Time: time.Now(), Valid: true},
		LastError:           errMsg,
	})
	if err != nil {
		slog.Warn("failed to update sync status", "error", err)
	}
}

// logEvent creates a network source event record.
func (s *Service) logEvent(ctx context.Context, sourceID uuid.UUID, filename, remotePath, action string, docID *uuid.UUID, errMsg string) {
	var pgDocID pgtype.UUID
	if docID != nil {
		pgDocID = pgtype.UUID{Bytes: *docID, Valid: true}
	}

	var errPtr *string
	if errMsg != "" {
		errPtr = &errMsg
	}

	_, err := s.db.Queries.CreateNetworkSourceEvent(ctx, sqlc.CreateNetworkSourceEventParams{
		SourceID:     sourceID,
		Filename:     filename,
		RemotePath:   remotePath,
		Action:       action,
		DocumentID:   pgDocID,
		ErrorMessage: errPtr,
	})
	if err != nil {
		slog.Warn("failed to log event", "error", err)
	}
}

// GetCrypto returns the credential crypto instance for use by handlers.
func (s *Service) GetCrypto() *CredentialCrypto {
	return s.crypto
}
