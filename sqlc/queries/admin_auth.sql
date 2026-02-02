-- name: GetAdminUserByUsername :one
SELECT * FROM admin_users WHERE username = $1 LIMIT 1;

-- name: CreateAdminUser :one
INSERT INTO admin_users (username, password_hash)
VALUES ($1, $2)
RETURNING *;

-- name: UpdateAdminUserPassword :exec
UPDATE admin_users
SET password_hash = $1, updated_at = NOW()
WHERE username = $2;

-- name: CreateAdminSession :one
INSERT INTO admin_sessions (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetAdminSessionByTokenHash :one
SELECT s.*, u.username
FROM admin_sessions s
JOIN admin_users u ON s.user_id = u.id
WHERE s.token_hash = $1 AND s.expires_at > NOW()
LIMIT 1;

-- name: DeleteAdminSession :exec
DELETE FROM admin_sessions WHERE token_hash = $1;

-- name: DeleteExpiredAdminSessions :exec
DELETE FROM admin_sessions WHERE expires_at <= NOW();

-- name: DeleteAdminUserSessions :exec
DELETE FROM admin_sessions WHERE user_id = $1;
