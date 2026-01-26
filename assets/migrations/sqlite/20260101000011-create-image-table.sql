-- +migrate Up
CREATE TABLE IF NOT EXISTS image (
    id TEXT PRIMARY KEY,
    site_id TEXT NOT NULL,
    short_id TEXT,
    file_name TEXT NOT NULL,
    file_path TEXT NOT NULL,
    alt_text TEXT,
    title TEXT,
    width INTEGER,
    height INTEGER,
    created_by TEXT,
    updated_by TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    FOREIGN KEY (site_id) REFERENCES site(id) ON DELETE CASCADE,
    UNIQUE(site_id, file_path)
);

CREATE INDEX IF NOT EXISTS idx_image_site_id ON image(site_id);

-- +migrate Down
DROP TABLE IF EXISTS image;
