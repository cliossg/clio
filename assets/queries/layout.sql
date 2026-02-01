-- name: CreateLayout :one
INSERT INTO layout (id, site_id, short_id, name, description, code, css, exclude_default_css, header_image_id, created_by, updated_by, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetLayout :one
SELECT * FROM layout WHERE id = ?;

-- name: GetLayoutByName :one
SELECT * FROM layout WHERE site_id = ? AND name = ?;

-- name: GetLayoutsBySiteID :many
SELECT * FROM layout WHERE site_id = ? ORDER BY name;

-- name: UpdateLayout :one
UPDATE layout SET
    name = ?,
    description = ?,
    code = ?,
    css = ?,
    exclude_default_css = ?,
    header_image_id = ?,
    updated_by = ?,
    updated_at = ?
WHERE id = ?
RETURNING *;

-- name: DeleteLayout :exec
DELETE FROM layout WHERE id = ?;
