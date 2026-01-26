-- +migrate Up
CREATE TABLE IF NOT EXISTS image_variant (
    id TEXT PRIMARY KEY,
    short_id TEXT NOT NULL DEFAULT '',
    image_id TEXT NOT NULL,
    kind TEXT NOT NULL,
    blob_ref TEXT NOT NULL,
    width INTEGER,
    height INTEGER,
    filesize_bytes INTEGER,
    mime TEXT,
    created_by TEXT NOT NULL DEFAULT '',
    updated_by TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    FOREIGN KEY (image_id) REFERENCES image(id) ON DELETE CASCADE,
    UNIQUE(image_id, kind)
);

CREATE INDEX IF NOT EXISTS idx_image_variant_image_id ON image_variant(image_id);

-- +migrate Down
DROP TABLE IF EXISTS image_variant;
