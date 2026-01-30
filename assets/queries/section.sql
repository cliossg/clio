-- name: CreateSection :one
INSERT INTO section (id, site_id, short_id, name, description, path, layout_id, layout_name, hero_title_dark, created_by, updated_by, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetSection :one
SELECT * FROM section WHERE id = ?;

-- name: GetSectionByPath :one
SELECT * FROM section WHERE site_id = ? AND path = ?;

-- name: GetSectionsBySiteID :many
SELECT * FROM section WHERE site_id = ? ORDER BY path;

-- name: GetSectionsWithHeaderImage :many
SELECT
    s.*,
    hi.file_path as header_image_path,
    hi.alt_text as header_image_alt
FROM section s
LEFT JOIN section_images si ON s.id = si.section_id AND si.is_header = 1
LEFT JOIN image hi ON si.image_id = hi.id
WHERE s.site_id = ?
ORDER BY s.path;

-- name: UpdateSection :one
UPDATE section SET
    name = ?,
    description = ?,
    path = ?,
    layout_id = ?,
    layout_name = ?,
    hero_title_dark = ?,
    updated_by = ?,
    updated_at = ?
WHERE id = ?
RETURNING *;

-- name: DeleteSection :exec
DELETE FROM section WHERE id = ?;
