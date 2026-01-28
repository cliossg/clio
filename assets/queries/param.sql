-- name: CreateParam :one
INSERT INTO param (id, site_id, short_id, name, description, value, ref_key, category, position, system, created_by, updated_by, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetParam :one
SELECT * FROM param WHERE id = ?;

-- name: GetParamByName :one
SELECT * FROM param WHERE site_id = ? AND name = ?;

-- name: GetParamByRefKey :one
SELECT * FROM param WHERE site_id = ? AND ref_key = ?;

-- name: GetParamsBySiteID :many
SELECT * FROM param WHERE site_id = ? ORDER BY category, position, name;

-- name: UpdateParam :one
UPDATE param SET
    name = ?,
    description = ?,
    value = ?,
    ref_key = ?,
    category = ?,
    position = ?,
    system = ?,
    updated_by = ?,
    updated_at = ?
WHERE id = ?
RETURNING *;

-- name: DeleteParam :exec
DELETE FROM param WHERE id = ?;
