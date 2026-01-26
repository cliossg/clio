-- +migrate Up
CREATE TABLE IF NOT EXISTS meta (
    id TEXT PRIMARY KEY,
    site_id TEXT NOT NULL,
    short_id TEXT,
    content_id TEXT NOT NULL,
    summary TEXT,
    excerpt TEXT,
    description TEXT,
    keywords TEXT,
    robots TEXT,
    canonical_url TEXT,
    sitemap TEXT,
    table_of_contents INTEGER DEFAULT 0,
    share INTEGER DEFAULT 0,
    comments INTEGER DEFAULT 0,
    created_by TEXT,
    updated_by TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    FOREIGN KEY (site_id) REFERENCES site(id) ON DELETE CASCADE,
    FOREIGN KEY (content_id) REFERENCES content(id) ON DELETE CASCADE,
    UNIQUE(content_id)
);

CREATE INDEX IF NOT EXISTS idx_meta_site_id ON meta(site_id);
CREATE INDEX IF NOT EXISTS idx_meta_content_id ON meta(content_id);

-- +migrate Down
DROP TABLE IF EXISTS meta;
