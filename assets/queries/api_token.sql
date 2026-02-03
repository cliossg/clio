-- name: CreateAPIToken :one
INSERT INTO api_token (id, user_id, name, token_hash, expires_at, created_at)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetAPITokenByHash :one
SELECT * FROM api_token WHERE token_hash = ?;

-- name: ListAPITokensByUser :many
SELECT * FROM api_token WHERE user_id = ? ORDER BY created_at DESC;

-- name: DeleteAPIToken :exec
DELETE FROM api_token WHERE id = ?;

-- name: UpdateAPITokenLastUsed :exec
UPDATE api_token SET last_used_at = ? WHERE id = ?;
