-- +migrate Up
CREATE TABLE IF NOT EXISTS site (
    id TEXT PRIMARY KEY,
    short_id TEXT NOT NULL DEFAULT '',
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    active INTEGER NOT NULL DEFAULT 1,
    default_layout_id TEXT,
    default_layout_name TEXT,
    created_by TEXT NOT NULL,
    updated_by TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_published_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_site_slug ON site(slug);
CREATE INDEX IF NOT EXISTS idx_site_active ON site(active);

-- +migrate Down
DROP TABLE IF EXISTS site;
