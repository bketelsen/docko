package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"docko/internal/database"
	"docko/internal/database/sqlc"
)

// JobHandler processes a job and returns an error if it fails
type JobHandler func(ctx context.Context, job *sqlc.Job) error

// Config holds queue configuration
type Config struct {
	PollInterval      time.Duration // How often to check for jobs (default: 1s)
	VisibilityTimeout time.Duration // How long a job stays invisible (default: 5min)
	WorkerCount       int           // Number of concurrent workers (default: 4)
	BaseRetryDelay    time.Duration // Base delay for exponential backoff (default: 1s)
	MaxRetryDelay     time.Duration // Maximum retry delay (default: 1h)
}

// DefaultConfig returns sensible defaults
func DefaultConfig() Config {
	return Config{
		PollInterval:      time.Second,
		VisibilityTimeout: 5 * time.Minute,
		WorkerCount:       4,
		BaseRetryDelay:    time.Second,
		MaxRetryDelay:     time.Hour,
	}
}

// Queue manages job processing
type Queue struct {
	db       *database.DB
	config   Config
	handlers map[string]JobHandler
	mu       sync.RWMutex
	wg       sync.WaitGroup
	stop     chan struct{}
	running  bool
}

// New creates a new Queue instance
func New(db *database.DB, config Config) *Queue {
	if config.PollInterval == 0 {
		config.PollInterval = time.Second
	}
	if config.WorkerCount == 0 {
		config.WorkerCount = 4
	}
	if config.BaseRetryDelay == 0 {
		config.BaseRetryDelay = time.Second
	}
	if config.MaxRetryDelay == 0 {
		config.MaxRetryDelay = time.Hour
	}

	return &Queue{
		db:       db,
		config:   config,
		handlers: make(map[string]JobHandler),
		stop:     make(chan struct{}),
	}
}

// RegisterHandler registers a handler for a job type
func (q *Queue) RegisterHandler(jobType string, handler JobHandler) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.handlers[jobType] = handler
}

// Enqueue adds a job to the queue
func (q *Queue) Enqueue(ctx context.Context, queueName, jobType string, payload any) (*sqlc.Job, error) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	job, err := q.db.Queries.EnqueueJob(ctx, sqlc.EnqueueJobParams{
		QueueName: queueName,
		JobType:   jobType,
		Payload:   payloadJSON,
		Column4:   nil, // use default max_attempts
		Column5:   pgtype.Timestamptz{}, // use default scheduled_at (NOW())
	})
	if err != nil {
		return nil, fmt.Errorf("enqueue job: %w", err)
	}

	slog.Info("job enqueued", "job_id", job.ID, "type", jobType, "queue", queueName)
	return &job, nil
}

// EnqueueTx adds a job within an existing transaction
func (q *Queue) EnqueueTx(ctx context.Context, qtx *sqlc.Queries, queueName, jobType string, payload any) (*sqlc.Job, error) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	job, err := qtx.EnqueueJob(ctx, sqlc.EnqueueJobParams{
		QueueName: queueName,
		JobType:   jobType,
		Payload:   payloadJSON,
		Column4:   nil, // use default max_attempts
		Column5:   pgtype.Timestamptz{}, // use default scheduled_at (NOW())
	})
	if err != nil {
		return nil, fmt.Errorf("enqueue job: %w", err)
	}

	return &job, nil
}

// Start begins processing jobs
func (q *Queue) Start(ctx context.Context, queueName string) {
	q.mu.Lock()
	if q.running {
		q.mu.Unlock()
		return
	}
	q.running = true
	q.mu.Unlock()

	slog.Info("queue starting", "queue", queueName, "workers", q.config.WorkerCount)

	for i := 0; i < q.config.WorkerCount; i++ {
		q.wg.Add(1)
		go q.worker(ctx, queueName, i)
	}
}

// Stop gracefully stops all workers
func (q *Queue) Stop() {
	q.mu.Lock()
	if !q.running {
		q.mu.Unlock()
		return
	}
	q.running = false
	q.mu.Unlock()

	close(q.stop)
	q.wg.Wait()
	slog.Info("queue stopped")
}

func (q *Queue) worker(ctx context.Context, queueName string, workerID int) {
	defer q.wg.Done()

	slog.Debug("worker started", "worker_id", workerID, "queue", queueName)

	ticker := time.NewTicker(q.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-q.stop:
			slog.Debug("worker stopping", "worker_id", workerID)
			return
		case <-ctx.Done():
			slog.Debug("worker context cancelled", "worker_id", workerID)
			return
		case <-ticker.C:
			q.processJobs(ctx, queueName, workerID)
		}
	}
}

func (q *Queue) processJobs(ctx context.Context, queueName string, workerID int) {
	// Dequeue one job at a time per worker
	limit := int64(1)
	jobs, err := q.db.Queries.DequeueJobs(ctx, sqlc.DequeueJobsParams{
		Column1: &queueName,
		Column2: &limit,
	})
	if err != nil {
		slog.Error("failed to dequeue jobs", "error", err, "worker_id", workerID)
		return
	}

	for _, job := range jobs {
		q.processJob(ctx, &job, workerID)
	}
}

func (q *Queue) processJob(ctx context.Context, job *sqlc.Job, workerID int) {
	q.mu.RLock()
	handler, ok := q.handlers[job.JobType]
	q.mu.RUnlock()

	if !ok {
		slog.Error("no handler for job type", "job_type", job.JobType, "job_id", job.ID)
		q.failJob(ctx, job.ID, fmt.Sprintf("no handler registered for job type: %s", job.JobType))
		return
	}

	slog.Info("processing job", "job_id", job.ID, "type", job.JobType, "attempt", job.Attempt, "worker_id", workerID)

	start := time.Now()
	err := handler(ctx, job)
	duration := time.Since(start)

	if err != nil {
		slog.Error("job failed", "job_id", job.ID, "error", err, "duration", duration, "attempt", job.Attempt)
		q.handleJobFailure(ctx, job, err)
		return
	}

	slog.Info("job completed", "job_id", job.ID, "duration", duration)
	if _, err := q.db.Queries.CompleteJob(ctx, job.ID); err != nil {
		slog.Error("failed to mark job complete", "job_id", job.ID, "error", err)
	}
}

func (q *Queue) handleJobFailure(ctx context.Context, job *sqlc.Job, jobErr error) {
	// Check if we've exhausted retries
	if job.Attempt >= job.MaxAttempts {
		slog.Warn("job exhausted retries", "job_id", job.ID, "attempts", job.Attempt)
		q.failJob(ctx, job.ID, jobErr.Error())
		return
	}

	// Schedule retry with exponential backoff + jitter
	delay := q.nextRetryDelay(job.Attempt)
	scheduledAt := time.Now().Add(delay)

	_, err := q.db.Queries.RetryJob(ctx, sqlc.RetryJobParams{
		ID: job.ID,
		ScheduledAt: pgtype.Timestamptz{
			Time:  scheduledAt,
			Valid: true,
		},
	})
	if err != nil {
		slog.Error("failed to schedule retry", "job_id", job.ID, "error", err)
		return
	}

	slog.Info("job scheduled for retry", "job_id", job.ID, "delay", delay, "scheduled_at", scheduledAt)
}

func (q *Queue) failJob(ctx context.Context, jobID uuid.UUID, errMsg string) {
	_, err := q.db.Queries.FailJob(ctx, sqlc.FailJobParams{
		ID:        jobID,
		LastError: &errMsg,
	})
	if err != nil {
		slog.Error("failed to mark job as failed", "job_id", jobID, "error", err)
	}
}

// nextRetryDelay calculates exponential backoff with full jitter
// Formula: random(0, min(cap, base * 2^attempt))
func (q *Queue) nextRetryDelay(attempt int32) time.Duration {
	backoff := float64(q.config.BaseRetryDelay) * math.Pow(2, float64(attempt))
	if backoff > float64(q.config.MaxRetryDelay) {
		backoff = float64(q.config.MaxRetryDelay)
	}
	jittered := rand.Float64() * backoff
	return time.Duration(jittered)
}
