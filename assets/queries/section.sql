-- name: CreateSection :one
INSERT INTO section (id, site_id, short_id, name, description, path, layout_id, layout_name, created_by, updated_by, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetSection :one
SELECT * FROM section WHERE id = ?;

-- name: GetSectionByPath :one
SELECT * FROM section WHERE site_id = ? AND path = ?;

-- name: GetSectionsBySiteID :many
SELECT * FROM section WHERE site_id = ? ORDER BY path;

-- name: UpdateSection :one
UPDATE section SET
    name = ?,
    description = ?,
    path = ?,
    layout_id = ?,
    layout_name = ?,
    updated_by = ?,
    updated_at = ?
WHERE id = ?
RETURNING *;

-- name: DeleteSection :exec
DELETE FROM section WHERE id = ?;
