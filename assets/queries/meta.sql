-- name: CreateMeta :one
INSERT INTO meta (id, site_id, short_id, content_id, summary, excerpt, description, keywords, robots, canonical_url, sitemap, table_of_contents, share, comments, created_by, updated_by, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetMeta :one
SELECT * FROM meta WHERE id = ?;

-- name: GetMetaByContentID :one
SELECT * FROM meta WHERE content_id = ?;

-- name: UpdateMeta :one
UPDATE meta SET
    summary = ?,
    excerpt = ?,
    description = ?,
    keywords = ?,
    robots = ?,
    canonical_url = ?,
    sitemap = ?,
    table_of_contents = ?,
    share = ?,
    comments = ?,
    updated_by = ?,
    updated_at = ?
WHERE id = ?
RETURNING *;

-- name: DeleteMeta :exec
DELETE FROM meta WHERE id = ?;

-- name: DeleteMetaByContentID :exec
DELETE FROM meta WHERE content_id = ?;
