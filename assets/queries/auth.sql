-- name: CreateUser :one
INSERT INTO user (id, short_id, email, password_hash, name, status, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetUser :one
SELECT * FROM user WHERE id = ?;

-- name: GetUserByEmail :one
SELECT * FROM user WHERE email = ?;

-- name: ListUsers :many
SELECT * FROM user ORDER BY created_at DESC;

-- name: UpdateUser :one
UPDATE user SET
    email = ?,
    password_hash = ?,
    name = ?,
    status = ?,
    updated_at = ?
WHERE id = ?
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM user WHERE id = ?;

-- name: CreateSession :one
INSERT INTO session (id, user_id, expires_at, created_at)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: GetSession :one
SELECT * FROM session WHERE id = ?;

-- name: GetValidSession :one
SELECT * FROM session WHERE id = ? AND expires_at > datetime('now');

-- name: DeleteSession :exec
DELETE FROM session WHERE id = ?;

-- name: DeleteExpiredSessions :exec
DELETE FROM session WHERE expires_at <= datetime('now');

-- name: DeleteUserSessions :exec
DELETE FROM session WHERE user_id = ?;
