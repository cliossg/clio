-- +migrate Up
CREATE TABLE IF NOT EXISTS layout (
    id TEXT PRIMARY KEY,
    site_id TEXT NOT NULL,
    short_id TEXT,
    name TEXT NOT NULL,
    description TEXT,
    code TEXT,
    header_image_id TEXT,
    created_by TEXT,
    updated_by TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    FOREIGN KEY (site_id) REFERENCES site(id) ON DELETE CASCADE,
    UNIQUE(site_id, name)
);

CREATE INDEX IF NOT EXISTS idx_layout_site_id ON layout(site_id);

-- +migrate Down
DROP TABLE IF EXISTS layout;
