-- name: CreateContributor :one
INSERT INTO contributor (id, short_id, site_id, handle, name, surname, bio, social_links, role, created_by, updated_by, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetContributor :one
SELECT * FROM contributor WHERE id = ?;

-- name: GetContributorByHandle :one
SELECT * FROM contributor WHERE site_id = ? AND handle = ?;

-- name: ListContributorsBySiteID :many
SELECT * FROM contributor WHERE site_id = ? ORDER BY name, surname;

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
