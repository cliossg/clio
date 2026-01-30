-- name: CreateSite :one
INSERT INTO site (id, short_id, name, slug, active, created_by, updated_by, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetSite :one
SELECT * FROM site WHERE id = ?;

-- name: GetSiteBySlug :one
SELECT * FROM site WHERE slug = ?;

-- name: ListSites :many
SELECT * FROM site WHERE active = 1 ORDER BY name;

-- name: ListAllSites :many
SELECT * FROM site ORDER BY name;

-- name: UpdateSite :one
UPDATE site SET
    name = ?,
    slug = ?,
    active = ?,
    default_layout_id = ?,
    default_layout_name = ?,
    updated_by = ?,
    updated_at = ?
WHERE id = ?
RETURNING *;

-- name: DeleteSite :exec
DELETE FROM site WHERE id = ?;
