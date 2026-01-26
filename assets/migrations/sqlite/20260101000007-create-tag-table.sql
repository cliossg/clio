-- +migrate Up
CREATE TABLE IF NOT EXISTS tag (
    id TEXT PRIMARY KEY,
    site_id TEXT NOT NULL,
    short_id TEXT,
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    created_by TEXT,
    updated_by TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    FOREIGN KEY (site_id) REFERENCES site(id) ON DELETE CASCADE,
    UNIQUE(site_id, slug)
);

CREATE INDEX IF NOT EXISTS idx_tag_site_id ON tag(site_id);
CREATE INDEX IF NOT EXISTS idx_tag_slug ON tag(site_id, slug);

-- +migrate Down
DROP TABLE IF EXISTS tag;
