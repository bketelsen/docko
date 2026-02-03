-- name: CreateNetworkSource :one
INSERT INTO network_sources (
    name, protocol, host, share_path, username, password_encrypted,
    enabled, continuous_sync, post_import_action, move_subfolder,
    duplicate_action, batch_size
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *;

-- name: GetNetworkSource :one
SELECT * FROM network_sources WHERE id = $1;

-- name: ListNetworkSources :many
SELECT * FROM network_sources ORDER BY created_at ASC;

-- name: ListEnabledNetworkSources :many
SELECT * FROM network_sources WHERE enabled = true ORDER BY created_at ASC;

-- name: ListContinuousSyncSources :many
SELECT * FROM network_sources
WHERE enabled = true AND continuous_sync = true
ORDER BY created_at ASC;

-- name: UpdateNetworkSource :one
UPDATE network_sources SET
    name = $2, protocol = $3, host = $4, share_path = $5,
    username = $6, password_encrypted = $7, enabled = $8,
    continuous_sync = $9, post_import_action = $10, move_subfolder = $11,
    duplicate_action = $12, batch_size = $13, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateNetworkSourceStatus :exec
UPDATE network_sources SET
    connection_state = $2, consecutive_failures = $3,
    last_sync_at = $4, last_error = $5, updated_at = NOW()
WHERE id = $1;

-- name: IncrementFilesImported :exec
UPDATE network_sources SET
    files_imported = files_imported + 1, updated_at = NOW()
WHERE id = $1;

-- name: ResetConsecutiveFailures :exec
UPDATE network_sources SET
    consecutive_failures = 0, connection_state = 'connected', updated_at = NOW()
WHERE id = $1;

-- name: IncrementConsecutiveFailures :one
UPDATE network_sources SET
    consecutive_failures = consecutive_failures + 1, updated_at = NOW()
WHERE id = $1
RETURNING consecutive_failures;

-- name: DisableNetworkSource :exec
UPDATE network_sources SET enabled = false, updated_at = NOW()
WHERE id = $1;

-- name: DeleteNetworkSource :exec
DELETE FROM network_sources WHERE id = $1;

-- name: CreateNetworkSourceEvent :one
INSERT INTO network_source_events (source_id, filename, remote_path, action, document_id, error_message)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListNetworkSourceEvents :many
SELECT * FROM network_source_events
WHERE source_id = $1
ORDER BY created_at DESC
LIMIT $2;
