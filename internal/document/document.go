package document

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"docko/internal/database"
	"docko/internal/database/sqlc"
	"docko/internal/queue"
	"docko/internal/storage"
)

// Event types for audit trail
const (
	EventIngested           = "ingested"
	EventHashed             = "hashed"
	EventDuplicateFound     = "duplicate_found"
	EventTextExtracted      = "text_extracted"
	EventThumbnailGenerated = "thumbnail_generated"
	EventFailed             = "failed"
)

// Queue names
const (
	QueueDefault = "default"
)

// Job types
const (
	JobTypeProcess = "process_document"
)

// IngestPayload is the job payload for document processing
type IngestPayload struct {
	DocumentID uuid.UUID `json:"document_id"`
}

// Service handles document operations
type Service struct {
	db      *database.DB
	storage *storage.Storage
	queue   *queue.Queue
}

// New creates a new document Service
func New(db *database.DB, storage *storage.Storage, queue *queue.Queue) *Service {
	return &Service{
		db:      db,
		storage: storage,
		queue:   queue,
	}
}

// Ingest stores a new document from a source file path
// Returns the document ID, or existing document ID if duplicate
func (s *Service) Ingest(ctx context.Context, sourcePath, originalFilename string) (*sqlc.Document, bool, error) {
	start := time.Now()
	docID := uuid.New()

	// Compute destination path
	destPath := s.storage.PathForUUID(storage.CategoryOriginals, docID, filepath.Ext(originalFilename))

	// Copy file and compute hash in single pass
	contentHash, fileSize, err := s.storage.CopyAndHash(destPath, sourcePath)
	if err != nil {
		return nil, false, fmt.Errorf("copy and hash file: %w", err)
	}

	// Check for duplicate
	existing, err := s.db.Queries.GetDocumentByHash(ctx, contentHash)
	if err == nil {
		// Duplicate found - clean up copied file and return existing
		s.storage.Delete(destPath)
		slog.Info("duplicate document detected", "existing_id", existing.ID, "hash", contentHash[:16]+"...")

		// Log duplicate event on existing document
		s.LogEvent(ctx, existing.ID, EventDuplicateFound, map[string]any{
			"attempted_filename": originalFilename,
			"source_path":        sourcePath,
		}, nil, time.Since(start))

		return &existing, true, nil
	}
	if err != pgx.ErrNoRows {
		// Unexpected error
		s.storage.Delete(destPath)
		return nil, false, fmt.Errorf("check duplicate: %w", err)
	}

	// Start transaction for document + job creation
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		s.storage.Delete(destPath)
		return nil, false, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.db.Queries.WithTx(tx)

	// Create document record
	doc, err := qtx.CreateDocument(ctx, sqlc.CreateDocumentParams{
		ID:               docID,
		OriginalFilename: originalFilename,
		ContentHash:      contentHash,
		FileSize:         fileSize,
		// page_count, pdf_title, pdf_author, pdf_created_at filled by processing job
	})
	if err != nil {
		s.storage.Delete(destPath)
		return nil, false, fmt.Errorf("create document: %w", err)
	}

	// Log ingested event
	eventPayload, _ := json.Marshal(map[string]any{
		"source_path": sourcePath,
		"dest_path":   destPath,
		"file_size":   fileSize,
		"hash":        contentHash,
	})
	_, err = qtx.CreateDocumentEvent(ctx, sqlc.CreateDocumentEventParams{
		DocumentID:   doc.ID,
		EventType:    EventIngested,
		Payload:      eventPayload,
		DurationMs:   intPtr(int32(time.Since(start).Milliseconds())),
	})
	if err != nil {
		s.storage.Delete(destPath)
		return nil, false, fmt.Errorf("create event: %w", err)
	}

	// Enqueue processing job
	_, err = s.queue.EnqueueTx(ctx, qtx, QueueDefault, JobTypeProcess, IngestPayload{
		DocumentID: doc.ID,
	})
	if err != nil {
		s.storage.Delete(destPath)
		return nil, false, fmt.Errorf("enqueue job: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		s.storage.Delete(destPath)
		return nil, false, fmt.Errorf("commit transaction: %w", err)
	}

	slog.Info("document ingested", "id", doc.ID, "filename", originalFilename, "size", fileSize, "hash", contentHash[:16]+"...")

	return &doc, false, nil
}

// GetByID retrieves a document by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*sqlc.Document, error) {
	doc, err := s.db.Queries.GetDocument(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get document: %w", err)
	}
	return &doc, nil
}

// GetByHash retrieves a document by content hash
func (s *Service) GetByHash(ctx context.Context, hash string) (*sqlc.Document, error) {
	doc, err := s.db.Queries.GetDocumentByHash(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("get document by hash: %w", err)
	}
	return &doc, nil
}

// GetEvents retrieves all events for a document
func (s *Service) GetEvents(ctx context.Context, id uuid.UUID) ([]sqlc.DocumentEvent, error) {
	events, err := s.db.Queries.GetDocumentEvents(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get events: %w", err)
	}
	return events, nil
}

// LogEvent creates an audit trail event for a document
func (s *Service) LogEvent(ctx context.Context, docID uuid.UUID, eventType string, payload map[string]any, errMsg *string, duration time.Duration) error {
	var payloadJSON []byte
	if payload != nil {
		payloadJSON, _ = json.Marshal(payload)
	}

	_, err := s.db.Queries.CreateDocumentEvent(ctx, sqlc.CreateDocumentEventParams{
		DocumentID:   docID,
		EventType:    eventType,
		Payload:      payloadJSON,
		ErrorMessage: errMsg,
		DurationMs:   intPtr(int32(duration.Milliseconds())),
	})
	if err != nil {
		return fmt.Errorf("create event: %w", err)
	}
	return nil
}

// OriginalPath returns the file path for a document's original file
func (s *Service) OriginalPath(doc *sqlc.Document) string {
	return s.storage.PathForUUID(storage.CategoryOriginals, doc.ID, filepath.Ext(doc.OriginalFilename))
}

// ThumbnailPath returns the file path for a document's thumbnail
func (s *Service) ThumbnailPath(doc *sqlc.Document) string {
	return s.storage.PathForUUID(storage.CategoryThumbnails, doc.ID, ".webp")
}

// TextPath returns the file path for a document's extracted text
func (s *Service) TextPath(doc *sqlc.Document) string {
	return s.storage.PathForUUID(storage.CategoryText, doc.ID, ".txt")
}

// FileExists checks if a file exists at the given path
func (s *Service) FileExists(path string) bool {
	return s.storage.FileExists(path)
}

func intPtr(i int32) *int32 {
	return &i
}
