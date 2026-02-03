-- name: ListTagsWithCounts :many
SELECT t.id, t.name, t.color, t.created_at, COUNT(dt.document_id)::int AS document_count
FROM tags t
LEFT JOIN document_tags dt ON t.id = dt.tag_id
GROUP BY t.id
ORDER BY t.name;

-- name: GetTag :one
SELECT * FROM tags WHERE id = $1;

-- name: CreateTag :one
INSERT INTO tags (name, color)
VALUES ($1, $2)
ON CONFLICT (name) DO NOTHING
RETURNING *;

-- name: UpdateTag :one
UPDATE tags
SET name = $2, color = $3
WHERE id = $1
RETURNING *;

-- name: DeleteTag :exec
DELETE FROM tags WHERE id = $1;

-- name: SearchTags :many
SELECT * FROM tags
WHERE name ILIKE $1
ORDER BY name
LIMIT 10;

-- name: GetDocumentTags :many
SELECT t.* FROM tags t
INNER JOIN document_tags dt ON dt.tag_id = t.id
WHERE dt.document_id = $1
ORDER BY t.name;

-- name: AddDocumentTag :exec
INSERT INTO document_tags (document_id, tag_id)
VALUES ($1, $2)
ON CONFLICT (document_id, tag_id) DO NOTHING;

-- name: RemoveDocumentTag :exec
DELETE FROM document_tags
WHERE document_id = $1 AND tag_id = $2;

-- name: SearchTagsExcludingDocument :many
SELECT t.* FROM tags t
WHERE t.name ILIKE $1
AND t.id NOT IN (
  SELECT tag_id FROM document_tags WHERE document_id = $2
)
ORDER BY t.name
LIMIT 10;
