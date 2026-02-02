-- name: CreateProfile :one
INSERT INTO profile (id, site_id, short_id, slug, name, surname, bio, social_links, photo_path, created_by, updated_by, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetProfile :one
SELECT * FROM profile WHERE id = ?;

-- name: GetProfileBySlug :one
SELECT * FROM profile WHERE site_id = ? AND slug = ?;

-- name: ListProfiles :many
SELECT * FROM profile WHERE site_id = ? ORDER BY name ASC;

-- name: UpdateProfile :one
UPDATE profile SET
    slug = ?,
    name = ?,
    surname = ?,
    bio = ?,
    social_links = ?,
    photo_path = ?,
    updated_by = ?,
    updated_at = ?
WHERE id = ?
RETURNING *;

-- name: DeleteProfile :exec
DELETE FROM profile WHERE id = ?;
