package processing

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"docko/internal/database"
	"docko/internal/database/sqlc"
	"docko/internal/document"
	"docko/internal/storage"
)

// Processor orchestrates document processing (text extraction + thumbnail generation)
type Processor struct {
	db       *database.DB
	docSvc   *document.Service
	textExt  *TextExtractor
	thumbGen *ThumbnailGenerator
}

// New creates a new Processor
func New(db *database.DB, docSvc *document.Service, store *storage.Storage, placeholderPath string) *Processor {
	// Get storage path for OCR volumes
	storagePath := store.BasePath()
	ocrInputPath := storagePath + "/ocr-input"
	ocrOutputPath := storagePath + "/ocr-output"

	return &Processor{
		db:       db,
		docSvc:   docSvc,
		textExt:  NewTextExtractor(ocrInputPath, ocrOutputPath),
		thumbGen: NewThumbnailGenerator(store, placeholderPath),
	}
}

// HandleJob processes a document (implements queue.JobHandler)
// Both text extraction AND thumbnail generation must succeed (all-or-nothing)
func (p *Processor) HandleJob(ctx context.Context, job *sqlc.Job) error {
	// Parse job payload
	var payload document.IngestPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	docID := payload.DocumentID
	start := time.Now()

	slog.Info("processing document",
		"doc_id", docID,
		"job_id", job.ID,
		"attempt", job.Attempt)

	// Get document from service
	doc, err := p.docSvc.GetByID(ctx, docID)
	if err != nil {
		return fmt.Errorf("get document: %w", err)
	}

	// Update status to 'processing'
	_, err = p.db.Queries.SetDocumentProcessingStatus(ctx, sqlc.SetDocumentProcessingStatusParams{
		ID:               docID,
		ProcessingStatus: sqlc.ProcessingStatusProcessing,
	})
	if err != nil {
		return fmt.Errorf("set processing status: %w", err)
	}

	// Get PDF path
	pdfPath := p.docSvc.OriginalPath(doc)

	// Extract text
	textStart := time.Now()
	text, method, err := p.textExt.Extract(ctx, pdfPath)
	if err != nil {
		// Check if this is the final attempt
		if job.Attempt >= job.MaxAttempts {
			return p.quarantine(ctx, docID, fmt.Sprintf("text extraction failed: %v", err))
		}
		return fmt.Errorf("extract text: %w", err)
	}
	textDuration := time.Since(textStart)

	slog.Info("text extracted",
		"doc_id", docID,
		"method", method,
		"length", len(text),
		"duration_ms", textDuration.Milliseconds())

	// Generate thumbnail
	thumbStart := time.Now()
	thumbPath, err := p.thumbGen.Generate(ctx, pdfPath, docID)
	if err != nil {
		// Check if this is the final attempt
		if job.Attempt >= job.MaxAttempts {
			return p.quarantine(ctx, docID, fmt.Sprintf("thumbnail generation failed: %v", err))
		}
		return fmt.Errorf("generate thumbnail: %w", err)
	}
	thumbDuration := time.Since(thumbStart)

	slog.Info("thumbnail generated",
		"doc_id", docID,
		"path", thumbPath,
		"duration_ms", thumbDuration.Milliseconds())

	// All-or-nothing transaction: update document with results
	tx, err := p.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := p.db.Queries.WithTx(tx)

	// Update document with extracted text and processing status
	_, err = qtx.UpdateDocumentProcessing(ctx, sqlc.UpdateDocumentProcessingParams{
		ID:                 docID,
		TextContent:        &text,
		ThumbnailGenerated: true,
		ProcessingStatus:   sqlc.ProcessingStatusCompleted,
		ProcessingError:    nil,
		ProcessedAt: pgtype.Timestamptz{
			Time:  time.Now(),
			Valid: true,
		},
	})
	if err != nil {
		return fmt.Errorf("update document processing: %w", err)
	}

	// Log success event
	eventPayload, _ := json.Marshal(map[string]any{
		"text_length":      len(text),
		"text_method":      method,
		"text_duration_ms": textDuration.Milliseconds(),
		"thumb_path":       thumbPath,
		"thumb_duration_ms": thumbDuration.Milliseconds(),
		"total_duration_ms": time.Since(start).Milliseconds(),
	})

	_, err = qtx.CreateDocumentEvent(ctx, sqlc.CreateDocumentEventParams{
		DocumentID: docID,
		EventType:  "processing_complete",
		Payload:    eventPayload,
		DurationMs: intPtr(int32(time.Since(start).Milliseconds())),
	})
	if err != nil {
		return fmt.Errorf("create event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	slog.Info("document processing complete",
		"doc_id", docID,
		"text_length", len(text),
		"text_method", method,
		"duration_ms", time.Since(start).Milliseconds())

	return nil
}

// quarantine moves a document to failed status after repeated failures
func (p *Processor) quarantine(ctx context.Context, docID uuid.UUID, reason string) error {
	slog.Warn("quarantining document",
		"doc_id", docID,
		"reason", reason)

	// Update document status to failed
	_, err := p.db.Queries.UpdateDocumentProcessing(ctx, sqlc.UpdateDocumentProcessingParams{
		ID:                 docID,
		TextContent:        nil,
		ThumbnailGenerated: false,
		ProcessingStatus:   sqlc.ProcessingStatusFailed,
		ProcessingError:    &reason,
		ProcessedAt: pgtype.Timestamptz{
			Time:  time.Now(),
			Valid: true,
		},
	})
	if err != nil {
		return fmt.Errorf("update document status: %w", err)
	}

	// Log quarantine event
	eventPayload, _ := json.Marshal(map[string]any{
		"reason": reason,
	})

	_, err = p.db.Queries.CreateDocumentEvent(ctx, sqlc.CreateDocumentEventParams{
		DocumentID:   docID,
		EventType:    "quarantined",
		Payload:      eventPayload,
		ErrorMessage: &reason,
	})
	if err != nil {
		slog.Error("failed to log quarantine event", "doc_id", docID, "error", err)
	}

	// Return nil so the job is marked as completed (we've handled the failure)
	return nil
}

func intPtr(i int32) *int32 {
	return &i
}
