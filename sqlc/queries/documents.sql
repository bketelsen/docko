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

-- name: ListDocumentsWithCorrespondent :many
SELECT d.*, c.id as correspondent_id, c.name as correspondent_name
FROM documents d
LEFT JOIN document_correspondents dc ON dc.document_id = d.id
LEFT JOIN correspondents c ON c.id = dc.correspondent_id
ORDER BY d.created_at DESC
LIMIT $1 OFFSET $2;

-- name: SearchDocuments :many
SELECT
    d.*,
    c.id as correspondent_id,
    c.name as correspondent_name,
    CASE WHEN sqlc.narg(query)::text IS NOT NULL AND sqlc.narg(query)::text != ''
         THEN ts_rank(d.search_vector, websearch_to_tsquery('english', sqlc.narg(query)::text))
         ELSE 0 END as rank,
    CASE WHEN sqlc.narg(query)::text IS NOT NULL AND sqlc.narg(query)::text != ''
         THEN ts_headline('english', COALESCE(d.text_content, ''), websearch_to_tsquery('english', sqlc.narg(query)::text),
              'MaxFragments=1, MaxWords=30, MinWords=15, StartSel=<mark>, StopSel=</mark>')
         ELSE '' END as headline
FROM documents d
LEFT JOIN document_correspondents dc ON dc.document_id = d.id
LEFT JOIN correspondents c ON c.id = dc.correspondent_id
WHERE
    -- Full-text search (optional - empty/null matches all)
    (sqlc.narg(query)::text IS NULL OR sqlc.narg(query)::text = ''
        OR d.search_vector @@ websearch_to_tsquery('english', sqlc.narg(query)::text))
    -- Correspondent filter (optional)
    AND (NOT sqlc.arg(has_correspondent)::boolean OR c.id = sqlc.arg(correspondent_id)::uuid)
    -- Date range filter (optional)
    AND (NOT sqlc.arg(has_date_from)::boolean OR d.document_date >= sqlc.arg(date_from)::timestamptz)
    AND (NOT sqlc.arg(has_date_to)::boolean OR d.document_date <= sqlc.arg(date_to)::timestamptz)
    -- Tag filter (optional - AND logic: must have ALL selected tags)
    AND (NOT sqlc.arg(has_tags)::boolean
        OR d.id IN (
            SELECT dt.document_id
            FROM document_tags dt
            WHERE dt.tag_id = ANY(sqlc.arg(tag_ids)::uuid[])
            GROUP BY dt.document_id
            HAVING COUNT(DISTINCT dt.tag_id) = sqlc.arg(tag_count)::int
        ))
ORDER BY
    CASE WHEN sqlc.narg(query)::text IS NOT NULL AND sqlc.narg(query)::text != ''
         THEN ts_rank(d.search_vector, websearch_to_tsquery('english', sqlc.narg(query)::text))
         ELSE 0 END DESC,
    d.document_date DESC NULLS LAST
LIMIT sqlc.arg(limit_count) OFFSET sqlc.arg(offset_count);

-- name: CountSearchDocuments :one
SELECT COUNT(*)::int as total
FROM documents d
LEFT JOIN document_correspondents dc ON dc.document_id = d.id
LEFT JOIN correspondents c ON c.id = dc.correspondent_id
WHERE
    (sqlc.narg(query)::text IS NULL OR sqlc.narg(query)::text = ''
        OR d.search_vector @@ websearch_to_tsquery('english', sqlc.narg(query)::text))
    AND (NOT sqlc.arg(has_correspondent)::boolean OR c.id = sqlc.arg(correspondent_id)::uuid)
    AND (NOT sqlc.arg(has_date_from)::boolean OR d.document_date >= sqlc.arg(date_from)::timestamptz)
    AND (NOT sqlc.arg(has_date_to)::boolean OR d.document_date <= sqlc.arg(date_to)::timestamptz)
    AND (NOT sqlc.arg(has_tags)::boolean
        OR d.id IN (
            SELECT dt.document_id
            FROM document_tags dt
            WHERE dt.tag_id = ANY(sqlc.arg(tag_ids)::uuid[])
            GROUP BY dt.document_id
            HAVING COUNT(DISTINCT dt.tag_id) = sqlc.arg(tag_count)::int
        ));
