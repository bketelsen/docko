package processing

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/bketelsen/docko/internal/ai"
	"github.com/bketelsen/docko/internal/database/sqlc"
)

// AIProcessor handles AI analysis jobs from the queue
type AIProcessor struct {
	aiSvc       *ai.Service
	broadcaster *StatusBroadcaster
}

// AIPayload is the job payload for AI analysis
type AIPayload struct {
	DocumentID uuid.UUID `json:"document_id"`
}

// JobTypeAI is the job type for AI analysis
const JobTypeAI = "ai_analyze"

// QueueAI is the queue name for AI jobs
const QueueAI = "ai"

// NewAIProcessor creates a new AI processor
func NewAIProcessor(aiSvc *ai.Service, broadcaster *StatusBroadcaster) *AIProcessor {
	return &AIProcessor{
		aiSvc:       aiSvc,
		broadcaster: broadcaster,
	}
}

// HandleJob processes an AI analysis job (implements queue.JobHandler)
func (p *AIProcessor) HandleJob(ctx context.Context, job *sqlc.Job) error {
	// Parse job payload
	var payload AIPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	docID := payload.DocumentID

	slog.Info("starting AI analysis",
		"doc_id", docID,
		"job_id", job.ID,
		"attempt", job.Attempt)

	// Broadcast AI processing status
	p.broadcast(StatusUpdate{
		DocumentID: docID,
		Status:     StatusAIProcessing,
		QueueName:  QueueAI,
	})

	// Run AI analysis
	jobID := job.ID
	result, err := p.aiSvc.AnalyzeDocument(ctx, docID, &jobID)
	if err != nil {
		slog.Error("AI analysis failed",
			"doc_id", docID,
			"job_id", job.ID,
			"error", err)

		// Broadcast failure status
		p.broadcast(StatusUpdate{
			DocumentID: docID,
			Status:     StatusAIFailed,
			Error:      err.Error(),
			QueueName:  QueueAI,
		})

		return fmt.Errorf("analyze document: %w", err)
	}

	slog.Info("AI analysis complete",
		"doc_id", docID,
		"job_id", job.ID,
		"provider", result.Provider,
		"auto_applied", result.AutoApplied,
		"pending", result.Pending,
		"skipped", result.Skipped,
		"duration_ms", result.Duration.Milliseconds())

	// Broadcast completion status
	p.broadcast(StatusUpdate{
		DocumentID: docID,
		Status:     StatusAIComplete,
		QueueName:  QueueAI,
	})

	return nil
}

// broadcast sends a status update to all SSE subscribers
func (p *AIProcessor) broadcast(update StatusUpdate) {
	if p.broadcaster != nil {
		p.broadcaster.Broadcast(update)
	}
}
