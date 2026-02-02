-- name: CreateInbox :one
INSERT INTO inboxes (path, name, error_path, duplicate_action, enabled)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetInbox :one
SELECT * FROM inboxes WHERE id = $1;

-- name: GetInboxByPath :one
SELECT * FROM inboxes WHERE path = $1;

-- name: ListInboxes :many
SELECT * FROM inboxes ORDER BY created_at ASC;

-- name: ListEnabledInboxes :many
SELECT * FROM inboxes WHERE enabled = true ORDER BY created_at ASC;

-- name: UpdateInbox :one
UPDATE inboxes
SET name = $2, path = $3, error_path = $4, duplicate_action = $5, enabled = $6, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateInboxStatus :exec
UPDATE inboxes
SET last_scan_at = $2, last_error = $3, updated_at = NOW()
WHERE id = $1;

-- name: DeleteInbox :exec
DELETE FROM inboxes WHERE id = $1;

-- name: CreateInboxEvent :one
INSERT INTO inbox_events (inbox_id, filename, action, document_id, error_message)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListInboxEvents :many
SELECT * FROM inbox_events
WHERE inbox_id = $1
ORDER BY created_at DESC
LIMIT $2;

-- name: ListRecentInboxEvents :many
SELECT ie.*, i.name as inbox_name
FROM inbox_events ie
JOIN inboxes i ON ie.inbox_id = i.id
ORDER BY ie.created_at DESC
LIMIT $1;
