-- AI Settings (singleton)

-- name: GetAISettings :one
SELECT * FROM ai_settings WHERE id = 1;

-- name: UpdateAISettings :one
UPDATE ai_settings SET
    preferred_provider = $1,
    max_pages = $2,
    auto_process = $3,
    auto_apply_threshold = $4,
    review_threshold = $5,
    updated_at = NOW()
WHERE id = 1
RETURNING *;

-- AI Suggestions

-- name: CreateAISuggestion :one
INSERT INTO ai_suggestions (
    document_id, job_id, suggestion_type, value, confidence, reasoning, is_new, status, resolved_at, resolved_by
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetAISuggestion :one
SELECT * FROM ai_suggestions WHERE id = $1;

-- name: ListDocumentSuggestions :many
SELECT * FROM ai_suggestions
WHERE document_id = $1
ORDER BY confidence DESC, created_at DESC;

-- name: ListPendingSuggestions :many
SELECT s.*, d.original_filename
FROM ai_suggestions s
JOIN documents d ON s.document_id = d.id
WHERE s.status = 'pending'
ORDER BY s.created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountPendingSuggestions :one
SELECT COUNT(*) FROM ai_suggestions WHERE status = 'pending';

-- name: ListPendingSuggestionsForDocument :many
SELECT * FROM ai_suggestions
WHERE document_id = $1 AND status = 'pending'
ORDER BY confidence DESC
LIMIT 5;

-- name: AcceptSuggestion :one
UPDATE ai_suggestions SET
    status = 'accepted',
    resolved_at = NOW(),
    resolved_by = 'user'
WHERE id = $1
RETURNING *;

-- name: RejectSuggestion :one
UPDATE ai_suggestions SET
    status = 'rejected',
    resolved_at = NOW(),
    resolved_by = 'user'
WHERE id = $1
RETURNING *;

-- name: AutoApplySuggestion :one
UPDATE ai_suggestions SET
    status = 'auto_applied',
    resolved_at = NOW(),
    resolved_by = 'auto'
WHERE id = $1
RETURNING *;

-- name: DeleteDocumentSuggestions :exec
DELETE FROM ai_suggestions WHERE document_id = $1;

-- AI Usage tracking

-- name: CreateAIUsage :one
INSERT INTO ai_usage (document_id, job_id, provider, model, input_tokens, output_tokens)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetAIUsageStats :one
SELECT
    COUNT(*) as documents_processed,
    COALESCE(SUM(input_tokens), 0) as total_input_tokens,
    COALESCE(SUM(output_tokens), 0) as total_output_tokens
FROM ai_usage;

-- name: GetAIUsageStatsByProvider :many
SELECT
    provider,
    COUNT(*) as request_count,
    COALESCE(SUM(input_tokens), 0) as total_input_tokens,
    COALESCE(SUM(output_tokens), 0) as total_output_tokens
FROM ai_usage
GROUP BY provider
ORDER BY request_count DESC;

-- name: GetRecentAIUsage :many
SELECT * FROM ai_usage
ORDER BY created_at DESC
LIMIT $1;
