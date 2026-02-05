package inbox

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/h2non/filetype"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/bketelsen/docko/internal/config"
	"github.com/bketelsen/docko/internal/database"
	"github.com/bketelsen/docko/internal/database/sqlc"
	"github.com/bketelsen/docko/internal/document"
)

// Inbox event action constants
const (
	ActionImported  = "imported"
	ActionDuplicate = "duplicate"
	ActionError     = "error"
	ActionInvalid   = "invalid"
)

// Default configuration values
const (
	DefaultDebounceDelay    = 500 * time.Millisecond
	DefaultMaxConcurrent    = 4
	DefaultErrorSubdir      = "errors"
)

// Service coordinates inbox watching and document ingestion.
type Service struct {
	db        *database.DB
	docSvc    *document.Service
	cfg       *config.Config
	watcher   *Watcher
	mu        sync.RWMutex
	watching  map[uuid.UUID]string // inbox ID -> path
	semaphore chan struct{}        // Limit concurrent ingestions
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// New creates a new inbox Service.
func New(db *database.DB, docSvc *document.Service, cfg *config.Config) *Service {
	return &Service{
		db:        db,
		docSvc:    docSvc,
		cfg:       cfg,
		watching:  make(map[uuid.UUID]string),
		semaphore: make(chan struct{}, DefaultMaxConcurrent),
	}
}

// Start initializes the watcher, loads inboxes, and begins watching.
func (s *Service) Start(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)

	// Create watcher with file handler
	watcher, err := NewWatcher(DefaultDebounceDelay, s.handleFile)
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}
	s.watcher = watcher

	// Create default inbox from config if needed
	if err := s.ensureDefaultInbox(ctx); err != nil {
		_ = s.watcher.Close()
		return fmt.Errorf("ensure default inbox: %w", err)
	}

	// Load and watch all enabled inboxes
	if err := s.RefreshInboxes(ctx); err != nil {
		_ = s.watcher.Close()
		return fmt.Errorf("refresh inboxes: %w", err)
	}

	// Scan existing files in all inboxes
	if err := s.scanAllInboxes(ctx); err != nil {
		slog.Warn("error scanning inboxes on startup", "error", err)
	}

	// Start watcher in background
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.watcher.Run(s.ctx); err != nil && err != context.Canceled {
			slog.Error("watcher error", "error", err)
		}
	}()

	slog.Info("inbox service started", "inboxes", len(s.watching))
	return nil
}

// Stop gracefully shuts down the inbox service.
func (s *Service) Stop() error {
	if s.cancel != nil {
		s.cancel()
	}
	if s.watcher != nil {
		_ = s.watcher.Close()
	}
	s.wg.Wait()
	slog.Info("inbox service stopped")
	return nil
}

// AddInbox starts watching a new inbox directory.
func (s *Service) AddInbox(inbox *sqlc.Inbox) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.watching[inbox.ID]; exists {
		return nil // Already watching
	}

	// Ensure directory exists
	if err := os.MkdirAll(inbox.Path, 0755); err != nil {
		return fmt.Errorf("create inbox directory: %w", err)
	}

	// Ensure error directory exists
	errorPath := s.getErrorPath(inbox)
	if err := os.MkdirAll(errorPath, 0755); err != nil {
		return fmt.Errorf("create error directory: %w", err)
	}

	if err := s.watcher.Add(inbox.Path); err != nil {
		return fmt.Errorf("watch directory: %w", err)
	}

	s.watching[inbox.ID] = inbox.Path
	return nil
}

// RemoveInbox stops watching an inbox directory.
func (s *Service) RemoveInbox(inboxID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path, exists := s.watching[inboxID]
	if !exists {
		return nil // Not watching
	}

	if err := s.watcher.Remove(path); err != nil {
		return fmt.Errorf("stop watching: %w", err)
	}

	delete(s.watching, inboxID)
	return nil
}

// RefreshInboxes reloads inboxes from database and updates watching state.
func (s *Service) RefreshInboxes(ctx context.Context) error {
	inboxes, err := s.db.Queries.ListEnabledInboxes(ctx)
	if err != nil {
		return fmt.Errorf("list enabled inboxes: %w", err)
	}

	// Build set of enabled inbox paths
	enabled := make(map[uuid.UUID]string)
	for _, inbox := range inboxes {
		enabled[inbox.ID] = inbox.Path
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove inboxes that are no longer enabled
	for id, path := range s.watching {
		if _, ok := enabled[id]; !ok {
			if err := s.watcher.Remove(path); err != nil {
				slog.Warn("failed to remove inbox", "id", id, "error", err)
			}
			delete(s.watching, id)
		}
	}

	// Add newly enabled inboxes
	for _, inbox := range inboxes {
		if _, exists := s.watching[inbox.ID]; exists {
			continue // Already watching
		}

		// Ensure directories exist
		if err := os.MkdirAll(inbox.Path, 0755); err != nil {
			slog.Warn("failed to create inbox directory", "path", inbox.Path, "error", err)
			continue
		}
		errorPath := s.getErrorPath(&inbox)
		if err := os.MkdirAll(errorPath, 0755); err != nil {
			slog.Warn("failed to create error directory", "path", errorPath, "error", err)
		}

		if err := s.watcher.Add(inbox.Path); err != nil {
			slog.Warn("failed to watch inbox", "path", inbox.Path, "error", err)
			continue
		}
		s.watching[inbox.ID] = inbox.Path
	}

	return nil
}

// ensureDefaultInbox creates the default inbox from config if INBOX_PATH is set
// and no inbox exists with that path.
func (s *Service) ensureDefaultInbox(ctx context.Context) error {
	if s.cfg.Inbox.DefaultPath == "" {
		return nil // No default path configured
	}

	// Check if inbox already exists
	_, err := s.db.Queries.GetInboxByPath(ctx, s.cfg.Inbox.DefaultPath)
	if err == nil {
		return nil // Already exists
	}
	if err != pgx.ErrNoRows {
		return fmt.Errorf("check existing inbox: %w", err)
	}

	// Create default inbox
	_, err = s.db.Queries.CreateInbox(ctx, sqlc.CreateInboxParams{
		Path:            s.cfg.Inbox.DefaultPath,
		Name:            "Default Inbox",
		Enabled:         true,
		DuplicateAction: sqlc.DuplicateActionDelete,
	})
	if err != nil {
		return fmt.Errorf("create default inbox: %w", err)
	}

	slog.Info("created default inbox", "path", s.cfg.Inbox.DefaultPath)
	return nil
}

// scanAllInboxes processes existing files in all watched inboxes.
func (s *Service) scanAllInboxes(ctx context.Context) error {
	inboxes, err := s.db.Queries.ListEnabledInboxes(ctx)
	if err != nil {
		return fmt.Errorf("list inboxes: %w", err)
	}

	for _, inbox := range inboxes {
		if err := s.scanDirectory(ctx, &inbox); err != nil {
			slog.Warn("failed to scan inbox", "path", inbox.Path, "error", err)
		}
	}
	return nil
}

// scanDirectory processes all PDF files in an inbox directory.
func (s *Service) scanDirectory(ctx context.Context, inbox *sqlc.Inbox) error {
	entries, err := os.ReadDir(inbox.Path)
	if err != nil {
		return fmt.Errorf("read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !isPDFFilename(entry.Name()) {
			continue
		}

		path := filepath.Join(inbox.Path, entry.Name())
		s.processFile(ctx, inbox, path)
	}

	// Update last scan time
	s.updateInboxStatus(ctx, inbox.ID, nil)
	return nil
}

// handleFile is called by the watcher when a file event stabilizes.
func (s *Service) handleFile(path string) {
	// Find inbox for this path
	inbox, err := s.findInboxForPath(path)
	if err != nil {
		slog.Warn("inbox not found for file", "path", path, "error", err)
		return
	}

	s.processFile(s.ctx, inbox, path)
}

// processFile handles a single file: validates, ingests, and cleans up.
func (s *Service) processFile(ctx context.Context, inbox *sqlc.Inbox, path string) {
	// Acquire semaphore slot
	s.semaphore <- struct{}{}
	defer func() { <-s.semaphore }()

	filename := filepath.Base(path)
	start := time.Now()

	slog.Debug("processing file", "path", path, "inbox", inbox.Name)

	// Validate PDF using magic bytes
	isPDF, err := s.validatePDF(path)
	if err != nil {
		slog.Warn("failed to validate file", "path", path, "error", err)
		s.handleError(ctx, inbox, path, filename, fmt.Sprintf("validation failed: %v", err))
		return
	}
	if !isPDF {
		slog.Info("file is not a valid PDF", "path", path)
		s.handleError(ctx, inbox, path, filename, "not a valid PDF file")
		return
	}

	// Ingest document
	doc, isDupe, err := s.docSvc.Ingest(ctx, path, filename)
	if err != nil {
		slog.Error("failed to ingest document", "path", path, "error", err)
		s.handleError(ctx, inbox, path, filename, fmt.Sprintf("ingestion failed: %v", err))
		return
	}

	// Handle result based on duplicate status
	if isDupe {
		s.handleDuplicate(ctx, inbox, path, filename, doc.ID)
	} else {
		s.handleSuccess(ctx, inbox, path, filename, doc.ID, time.Since(start))
	}
}

// validatePDF checks if a file is a valid PDF using magic bytes.
func (s *Service) validatePDF(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer func() { _ = f.Close() }()

	// Read first 262 bytes for magic byte detection
	buf := make([]byte, 262)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return false, err
	}

	return filetype.Is(buf[:n], "pdf"), nil
}

// handleSuccess processes a successfully ingested file.
func (s *Service) handleSuccess(ctx context.Context, inbox *sqlc.Inbox, path, filename string, docID uuid.UUID, duration time.Duration) {
	// Delete the source file
	if err := os.Remove(path); err != nil {
		slog.Warn("failed to delete imported file", "path", path, "error", err)
	}

	// Log inbox event
	s.logInboxEvent(ctx, inbox.ID, filename, ActionImported, &docID, nil)
	s.updateInboxStatus(ctx, inbox.ID, nil)

	slog.Info("file imported",
		"filename", filename,
		"document_id", docID,
		"inbox", inbox.Name,
		"duration_ms", duration.Milliseconds(),
	)
}

// handleDuplicate processes a duplicate file based on inbox settings.
func (s *Service) handleDuplicate(ctx context.Context, inbox *sqlc.Inbox, path, filename string, existingDocID uuid.UUID) {
	switch inbox.DuplicateAction {
	case sqlc.DuplicateActionDelete:
		// Delete silently, log the occurrence
		if err := os.Remove(path); err != nil {
			slog.Warn("failed to delete duplicate file", "path", path, "error", err)
		}
		s.logInboxEvent(ctx, inbox.ID, filename, ActionDuplicate, &existingDocID, nil)
		slog.Info("duplicate file deleted", "filename", filename, "existing_document_id", existingDocID)

	case sqlc.DuplicateActionRename:
		// Rename with timestamp suffix
		newPath := s.generateUniqueFilename(path)
		if err := os.Rename(path, newPath); err != nil {
			slog.Warn("failed to rename duplicate file", "path", path, "error", err)
			return
		}
		errMsg := "duplicate - renamed"
		s.logInboxEvent(ctx, inbox.ID, filename, ActionDuplicate, &existingDocID, &errMsg)
		slog.Info("duplicate file renamed", "filename", filename, "new_path", newPath)

	case sqlc.DuplicateActionSkip:
		// Leave file in place, log the occurrence
		errMsg := "duplicate - skipped"
		s.logInboxEvent(ctx, inbox.ID, filename, ActionDuplicate, &existingDocID, &errMsg)
		slog.Info("duplicate file skipped", "filename", filename)
	}
}

// handleError moves a failed file to the error directory.
func (s *Service) handleError(ctx context.Context, inbox *sqlc.Inbox, path, filename, errMsg string) {
	errorPath := s.getErrorPath(inbox)

	// Generate unique filename in error directory
	destPath := s.generateUniqueFilename(filepath.Join(errorPath, filename))

	// Move file to error directory
	if err := os.Rename(path, destPath); err != nil {
		slog.Error("failed to move file to error directory",
			"source", path,
			"dest", destPath,
			"error", err,
		)
		return
	}

	// Log inbox event
	s.logInboxEvent(ctx, inbox.ID, filename, ActionError, nil, &errMsg)
	s.updateInboxStatus(ctx, inbox.ID, &errMsg)

	slog.Info("file moved to error directory",
		"filename", filename,
		"dest", destPath,
		"error", errMsg,
	)
}

// getErrorPath returns the error directory path for an inbox.
func (s *Service) getErrorPath(inbox *sqlc.Inbox) string {
	if inbox.ErrorPath != nil && *inbox.ErrorPath != "" {
		return *inbox.ErrorPath
	}
	return filepath.Join(inbox.Path, s.cfg.Inbox.ErrorSubdir)
}

// generateUniqueFilename adds a timestamp to avoid filename collisions.
func (s *Service) generateUniqueFilename(path string) string {
	dir := filepath.Dir(path)
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(filepath.Base(path), ext)
	timestamp := time.Now().Format("20060102-150405")
	return filepath.Join(dir, fmt.Sprintf("%s_%s%s", base, timestamp, ext))
}

// findInboxForPath looks up the inbox that contains the given file path.
func (s *Service) findInboxForPath(path string) (*sqlc.Inbox, error) {
	dir := filepath.Dir(path)

	s.mu.RLock()
	var inboxID uuid.UUID
	found := false
	for id, inboxPath := range s.watching {
		if inboxPath == dir {
			inboxID = id
			found = true
			break
		}
	}
	s.mu.RUnlock()

	if !found {
		return nil, fmt.Errorf("no inbox watching directory: %s", dir)
	}

	inbox, err := s.db.Queries.GetInbox(s.ctx, inboxID)
	if err != nil {
		return nil, fmt.Errorf("get inbox: %w", err)
	}

	return &inbox, nil
}

// logInboxEvent creates an inbox event record.
func (s *Service) logInboxEvent(ctx context.Context, inboxID uuid.UUID, filename, action string, docID *uuid.UUID, errMsg *string) {
	var pgDocID pgtype.UUID
	if docID != nil {
		pgDocID = pgtype.UUID{Bytes: *docID, Valid: true}
	}

	_, err := s.db.Queries.CreateInboxEvent(ctx, sqlc.CreateInboxEventParams{
		InboxID:      inboxID,
		Filename:     filename,
		Action:       action,
		DocumentID:   pgDocID,
		ErrorMessage: errMsg,
	})
	if err != nil {
		slog.Warn("failed to log inbox event", "error", err)
	}
}

// updateInboxStatus updates the inbox's last_scan_at and optionally last_error.
func (s *Service) updateInboxStatus(ctx context.Context, inboxID uuid.UUID, errMsg *string) {
	err := s.db.Queries.UpdateInboxStatus(ctx, sqlc.UpdateInboxStatusParams{
		ID:         inboxID,
		LastScanAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		LastError:  errMsg,
	})
	if err != nil {
		slog.Warn("failed to update inbox status", "error", err)
	}
}
