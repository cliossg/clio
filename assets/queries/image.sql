-- name: CreateImage :one
INSERT INTO image (id, site_id, short_id, file_name, file_path, alt_text, title, width, height, created_by, updated_by, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetImage :one
SELECT * FROM image WHERE id = ?;

-- name: GetImageByShortID :one
SELECT * FROM image WHERE short_id = ?;

-- name: GetImageByPath :one
SELECT * FROM image WHERE site_id = ? AND file_path = ?;

-- name: GetImagesBySiteID :many
SELECT * FROM image WHERE site_id = ? ORDER BY created_at DESC;

-- name: UpdateImage :one
UPDATE image SET
    file_name = ?,
    file_path = ?,
    alt_text = ?,
    title = ?,
    width = ?,
    height = ?,
    updated_by = ?,
    updated_at = ?
WHERE id = ?
RETURNING *;

-- name: DeleteImage :exec
DELETE FROM image WHERE id = ?;

-- name: CreateImageVariant :one
INSERT INTO image_variant (id, short_id, image_id, kind, blob_ref, width, height, filesize_bytes, mime, created_by, updated_by, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetImageVariant :one
SELECT * FROM image_variant WHERE id = ?;

-- name: GetImageVariantsByImageID :many
SELECT * FROM image_variant WHERE image_id = ? ORDER BY kind;

-- name: UpdateImageVariant :one
UPDATE image_variant SET
    kind = ?,
    blob_ref = ?,
    width = ?,
    height = ?,
    filesize_bytes = ?,
    mime = ?,
    updated_by = ?,
    updated_at = ?
WHERE id = ?
RETURNING *;

-- name: DeleteImageVariant :exec
DELETE FROM image_variant WHERE id = ?;

-- name: CreateContentImage :exec
INSERT INTO content_images (id, content_id, image_id, is_header, is_featured, order_num, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetContentImagesByContentID :many
SELECT * FROM content_images WHERE content_id = ? ORDER BY order_num;

-- name: GetContentImagesWithDetails :many
SELECT
    ci.id as content_image_id,
    ci.content_id,
    ci.is_header,
    ci.is_featured,
    ci.order_num,
    i.id,
    i.site_id,
    i.short_id,
    i.file_name,
    i.file_path,
    i.alt_text,
    i.title,
    i.width,
    i.height,
    i.created_at,
    i.updated_at
FROM content_images ci
JOIN image i ON ci.image_id = i.id
WHERE ci.content_id = ?
ORDER BY ci.is_header DESC, ci.order_num;

-- name: GetContentImageWithDetails :one
SELECT
    ci.id as content_image_id,
    ci.content_id,
    ci.image_id,
    ci.is_header,
    i.id,
    i.site_id,
    i.file_path
FROM content_images ci
JOIN image i ON ci.image_id = i.id
WHERE ci.id = ?;

-- name: DeleteContentImage :exec
DELETE FROM content_images WHERE id = ?;

-- name: DeleteContentImageByContentAndImage :exec
DELETE FROM content_images WHERE content_id = ? AND image_id = ?;

-- name: CreateSectionImage :exec
INSERT INTO section_images (id, section_id, image_id, is_header, is_featured, order_num, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetSectionImagesBySectionID :many
SELECT * FROM section_images WHERE section_id = ? ORDER BY order_num;

-- name: DeleteSectionImage :exec
DELETE FROM section_images WHERE id = ?;
