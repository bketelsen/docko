-- name: CreateDocument :one
INSERT INTO documents (original_filename, content_hash, file_size, page_count, pdf_title, pdf_author, pdf_created_at, document_date)
VALUES ($1, $2, $3, $4, $5, $6, $7, COALESCE($8, NOW()))
RETURNING *;

-- name: GetDocument :one
SELECT * FROM documents WHERE id = $1;

-- name: GetDocumentByHash :one
SELECT * FROM documents WHERE content_hash = $1;

-- name: ListDocuments :many
SELECT * FROM documents ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: UpdateDocument :one
UPDATE documents SET
  document_date = COALESCE($2, document_date),
  updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteDocument :exec
DELETE FROM documents WHERE id = $1;

-- name: CreateDocumentEvent :one
INSERT INTO document_events (document_id, event_type, payload, error_message, duration_ms)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetDocumentEvents :many
SELECT * FROM document_events WHERE document_id = $1 ORDER BY created_at DESC;

-- name: GetLatestDocumentEvent :one
SELECT * FROM document_events WHERE document_id = $1 ORDER BY created_at DESC LIMIT 1;
