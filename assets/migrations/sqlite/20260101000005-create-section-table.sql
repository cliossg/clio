-- +migrate Up
CREATE TABLE IF NOT EXISTS section (
    id TEXT PRIMARY KEY,
    site_id TEXT NOT NULL,
    short_id TEXT,
    name TEXT NOT NULL,
    description TEXT,
    path TEXT,
    layout_id TEXT,
    layout_name TEXT,
    hero_title_dark INTEGER DEFAULT 0,
    created_by TEXT,
    updated_by TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    FOREIGN KEY (site_id) REFERENCES site(id) ON DELETE CASCADE,
    FOREIGN KEY (layout_id) REFERENCES layout(id),
    UNIQUE(site_id, path)
);

CREATE INDEX IF NOT EXISTS idx_section_site_id ON section(site_id);
CREATE INDEX IF NOT EXISTS idx_section_path ON section(site_id, path);

-- +migrate Down
DROP TABLE IF EXISTS section;
