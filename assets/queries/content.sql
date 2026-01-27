-- name: CreateContent :one
INSERT INTO content (id, site_id, user_id, short_id, section_id, contributor_id, kind, heading, summary, body, draft, featured, series, series_order, published_at, created_by, updated_by, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetContent :one
SELECT * FROM content WHERE id = ?;

-- name: GetContentBySiteID :many
SELECT * FROM content WHERE site_id = ? ORDER BY created_at DESC;

-- name: GetContentBySectionID :many
SELECT * FROM content WHERE section_id = ? ORDER BY created_at DESC;

-- name: GetPublishedContentBySiteID :many
SELECT * FROM content WHERE site_id = ? AND draft = 0 ORDER BY published_at DESC;

-- name: GetContentWithMeta :one
SELECT
    c.*,
    s.path as section_path,
    s.name as section_name,
    m.summary as meta_summary,
    m.description as meta_description,
    m.keywords as meta_keywords
FROM content c
LEFT JOIN section s ON c.section_id = s.id
LEFT JOIN meta m ON c.id = m.content_id
WHERE c.id = ?;

-- name: GetAllContentWithMeta :many
SELECT
    c.*,
    s.path as section_path,
    s.name as section_name,
    m.summary as meta_summary,
    m.description as meta_description,
    m.keywords as meta_keywords
FROM content c
LEFT JOIN section s ON c.section_id = s.id
LEFT JOIN meta m ON c.id = m.content_id
WHERE c.site_id = ?
ORDER BY c.created_at DESC;

-- name: GetContentWithPagination :many
SELECT * FROM content
WHERE site_id = ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: CountContent :one
SELECT COUNT(*) FROM content WHERE site_id = ?;

-- name: SearchContent :many
SELECT * FROM content
WHERE site_id = ? AND heading LIKE ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: CountSearchContent :one
SELECT COUNT(*) FROM content WHERE site_id = ? AND heading LIKE ?;

-- name: UpdateContent :one
UPDATE content SET
    section_id = ?,
    contributor_id = ?,
    kind = ?,
    heading = ?,
    summary = ?,
    body = ?,
    draft = ?,
    featured = ?,
    series = ?,
    series_order = ?,
    published_at = ?,
    updated_by = ?,
    updated_at = ?
WHERE id = ?
RETURNING *;

-- name: DeleteContent :exec
DELETE FROM content WHERE id = ?;
