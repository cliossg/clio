-- +migrate Up
CREATE TABLE IF NOT EXISTS param (
    id TEXT PRIMARY KEY,
    site_id TEXT NOT NULL,
    short_id TEXT,
    name TEXT NOT NULL,
    description TEXT,
    value TEXT,
    ref_key TEXT,
    system INTEGER DEFAULT 0,
    created_by TEXT,
    updated_by TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    FOREIGN KEY (site_id) REFERENCES site(id) ON DELETE CASCADE,
    UNIQUE(site_id, name)
);

CREATE INDEX IF NOT EXISTS idx_param_site_id ON param(site_id);
CREATE INDEX IF NOT EXISTS idx_param_name ON param(site_id, name);

-- +migrate Down
DROP TABLE IF EXISTS param;
