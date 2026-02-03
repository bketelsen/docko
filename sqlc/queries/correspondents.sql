-- name: ListCorrespondentsWithCounts :many
SELECT c.id, c.name, c.notes, c.created_at, COUNT(dc.document_id)::int AS document_count
FROM correspondents c
LEFT JOIN document_correspondents dc ON dc.correspondent_id = c.id
GROUP BY c.id
ORDER BY c.name;

-- name: GetCorrespondent :one
SELECT id, name, notes, created_at FROM correspondents WHERE id = $1;

-- name: CreateCorrespondent :one
INSERT INTO correspondents (name, notes)
VALUES ($1, $2)
RETURNING id, name, notes, created_at;

-- name: UpdateCorrespondent :one
UPDATE correspondents SET name = $1, notes = $2 WHERE id = $3
RETURNING id, name, notes, created_at;

-- name: DeleteCorrespondent :exec
DELETE FROM correspondents WHERE id = $1;

-- name: SearchCorrespondents :many
SELECT id, name, notes, created_at FROM correspondents
WHERE name ILIKE $1
ORDER BY name
LIMIT 10;

-- name: MergeCorrespondentsUpdateDocs :exec
UPDATE document_correspondents
SET correspondent_id = $1
WHERE correspondent_id = ANY($2::uuid[]);

-- name: GetCorrespondentsNotes :many
SELECT id, name, notes FROM correspondents
WHERE id = ANY($1::uuid[]) AND notes IS NOT NULL AND notes != '';

-- name: AppendCorrespondentNotes :one
UPDATE correspondents
SET notes = CASE
    WHEN notes IS NULL OR notes = '' THEN $2
    ELSE notes || E'\n---\n' || $2
END
WHERE id = $1
RETURNING *;

-- name: DeleteCorrespondentsByIds :exec
DELETE FROM correspondents WHERE id = ANY($1::uuid[]);
