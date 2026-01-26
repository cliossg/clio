-- name: CreateTag :one
INSERT INTO tag (id, site_id, short_id, name, slug, created_by, updated_by, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetTag :one
SELECT * FROM tag WHERE id = ?;

-- name: GetTagBySlug :one
SELECT * FROM tag WHERE site_id = ? AND slug = ?;

-- name: GetTagByName :one
SELECT * FROM tag WHERE site_id = ? AND name = ?;

-- name: GetTagsBySiteID :many
SELECT * FROM tag WHERE site_id = ? ORDER BY name;

-- name: UpdateTag :one
UPDATE tag SET
    name = ?,
    slug = ?,
    updated_by = ?,
    updated_at = ?
WHERE id = ?
RETURNING *;

-- name: DeleteTag :exec
DELETE FROM tag WHERE id = ?;

-- name: AddTagToContent :exec
INSERT INTO content_tag (id, content_id, tag_id, created_at)
VALUES (?, ?, ?, ?);

-- name: RemoveTagFromContent :exec
DELETE FROM content_tag WHERE content_id = ? AND tag_id = ?;

-- name: RemoveAllTagsFromContent :exec
DELETE FROM content_tag WHERE content_id = ?;

-- name: GetTagsForContent :many
SELECT t.* FROM tag t
JOIN content_tag ct ON t.id = ct.tag_id
WHERE ct.content_id = ?
ORDER BY t.name;

-- name: GetContentForTag :many
SELECT c.* FROM content c
JOIN content_tag ct ON c.id = ct.content_id
WHERE ct.tag_id = ?
ORDER BY c.created_at DESC;
