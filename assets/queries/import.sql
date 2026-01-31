-- name: CreateImport :one
INSERT INTO import (id, short_id, file_path, file_hash, file_mtime, content_id, site_id, user_id, status, imported_at, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetImport :one
SELECT * FROM import WHERE id = ?;

-- name: GetImportByFilePath :one
SELECT * FROM import WHERE file_path = ?;

-- name: GetImportByContentID :one
SELECT * FROM import WHERE content_id = ?;

-- name: ListImportsBySiteID :many
SELECT
    i.*,
    c.heading as content_heading,
    c.updated_at as content_updated_at
FROM import i
LEFT JOIN content c ON i.content_id = c.id
WHERE i.site_id = ?
ORDER BY i.created_at DESC;

-- name: UpdateImport :one
UPDATE import SET
    file_hash = ?,
    file_mtime = ?,
    content_id = ?,
    status = ?,
    imported_at = ?,
    updated_at = ?
WHERE id = ?
RETURNING *;

-- name: UpdateImportStatus :one
UPDATE import SET
    status = ?,
    updated_at = ?
WHERE id = ?
RETURNING *;

-- name: DeleteImport :exec
DELETE FROM import WHERE id = ?;

-- name: DeleteImportByContentID :exec
DELETE FROM import WHERE content_id = ?;
