-- name: CreateContributor :one
INSERT INTO contributor (id, short_id, site_id, profile_id, handle, name, surname, bio, social_links, role, created_by, updated_by, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetContributor :one
SELECT * FROM contributor WHERE id = ?;

-- name: GetContributorByHandle :one
SELECT * FROM contributor WHERE site_id = ? AND handle = ?;

-- name: ListContributorsBySiteID :many
SELECT * FROM contributor WHERE site_id = ? ORDER BY name, surname;

-- name: ListContributorsWithProfile :many
SELECT c.*, p.photo_path as profile_photo_path
FROM contributor c
LEFT JOIN profile p ON c.profile_id = p.id
WHERE c.site_id = ?
ORDER BY c.name, c.surname;

-- name: UpdateContributor :one
UPDATE contributor SET
    handle = ?,
    name = ?,
    surname = ?,
    bio = ?,
    social_links = ?,
    role = ?,
    updated_by = ?,
    updated_at = ?
WHERE id = ?
RETURNING *;

-- name: DeleteContributor :exec
DELETE FROM contributor WHERE id = ?;

-- name: SetContributorProfile :exec
UPDATE contributor SET profile_id = ?, updated_by = ?, updated_at = ? WHERE id = ?;
