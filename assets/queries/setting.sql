-- name: CreateSetting :one
INSERT INTO setting (id, site_id, short_id, name, description, value, ref_key, category, position, system, created_by, updated_by, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetSetting :one
SELECT * FROM setting WHERE id = ?;

-- name: GetSettingByName :one
SELECT * FROM setting WHERE site_id = ? AND name = ?;

-- name: GetSettingByRefKey :one
SELECT * FROM setting WHERE site_id = ? AND ref_key = ?;

-- name: GetSettingsBySiteID :many
SELECT * FROM setting WHERE site_id = ? ORDER BY category, position, name;

-- name: UpdateSetting :one
UPDATE setting SET
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

-- name: DeleteSetting :exec
DELETE FROM setting WHERE id = ?;
