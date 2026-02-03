-- name: CreateDocument :one
INSERT INTO documents (id, original_filename, content_hash, file_size, page_count, pdf_title, pdf_author, pdf_created_at, document_date)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, COALESCE($9, NOW()))
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

-- name: UpdateDocumentProcessing :one
UPDATE documents SET
    text_content = $2,
    thumbnail_generated = $3,
    processing_status = $4,
    processing_error = $5,
    processed_at = $6,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetPendingProcessingDocuments :many
SELECT * FROM documents
WHERE processing_status = 'pending'
ORDER BY created_at ASC
LIMIT $1;

-- name: SetDocumentProcessingStatus :one
UPDATE documents SET
    processing_status = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;
