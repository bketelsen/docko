-- name: EnqueueJob :one
INSERT INTO jobs (queue_name, job_type, payload, max_attempts, scheduled_at)
VALUES ($1, $2, $3, COALESCE($4, 3), COALESCE($5, NOW()))
RETURNING *;

-- name: DequeueJobs :many
WITH next_jobs AS (
    SELECT id FROM jobs
    WHERE queue_name = $1
      AND (status = 'pending' OR (status = 'processing' AND visible_until < NOW()))
      AND scheduled_at <= NOW()
      AND attempt < max_attempts
    ORDER BY created_at
    LIMIT $2
    FOR UPDATE SKIP LOCKED
)
UPDATE jobs
SET status = 'processing',
    attempt = attempt + 1,
    started_at = NOW(),
    visible_until = NOW() + INTERVAL '5 minutes',
    updated_at = NOW()
FROM next_jobs
WHERE jobs.id = next_jobs.id
RETURNING jobs.*;

-- name: CompleteJob :one
UPDATE jobs
SET status = 'completed',
    completed_at = NOW(),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: FailJob :one
UPDATE jobs
SET status = 'failed',
    last_error = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: RetryJob :one
UPDATE jobs
SET status = 'pending',
    scheduled_at = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetJob :one
SELECT * FROM jobs WHERE id = $1;

-- name: GetPendingJobCount :one
SELECT COUNT(*) FROM jobs WHERE queue_name = $1 AND status = 'pending';

-- name: GetFailedJobs :many
SELECT * FROM jobs WHERE queue_name = $1 AND status = 'failed' ORDER BY updated_at DESC LIMIT $2;
