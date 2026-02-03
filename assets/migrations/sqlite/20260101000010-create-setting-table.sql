-- +migrate Up
CREATE TABLE IF NOT EXISTS setting (
    id TEXT PRIMARY KEY,
    site_id TEXT NOT NULL,
    short_id TEXT,
    name TEXT NOT NULL,
    description TEXT,
    value TEXT,
    ref_key TEXT,
    system INTEGER DEFAULT 0,
    category TEXT DEFAULT '',
    position INTEGER DEFAULT 0,
    created_by TEXT,
    updated_by TEXT,
    type TEXT DEFAULT 'string',
    constraints TEXT,
    ui_control TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    FOREIGN KEY (site_id) REFERENCES site(id) ON DELETE CASCADE,
    UNIQUE(site_id, name)
);

CREATE INDEX IF NOT EXISTS idx_setting_site_id ON setting(site_id);
CREATE INDEX IF NOT EXISTS idx_setting_name ON setting(site_id, name);

-- +migrate Down
DROP TABLE IF EXISTS setting;
