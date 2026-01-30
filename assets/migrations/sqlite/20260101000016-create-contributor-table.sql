-- +migrate Up
CREATE TABLE contributor (
    id TEXT PRIMARY KEY,
    short_id TEXT NOT NULL,
    site_id TEXT NOT NULL REFERENCES site(id) ON DELETE CASCADE,
    profile_id TEXT REFERENCES profile(id) ON DELETE SET NULL,
    handle TEXT NOT NULL,
    name TEXT NOT NULL,
    surname TEXT NOT NULL DEFAULT '',
    bio TEXT NOT NULL DEFAULT '',
    social_links TEXT NOT NULL DEFAULT '[]',
    role TEXT NOT NULL DEFAULT 'editor',
    created_by TEXT NOT NULL,
    updated_by TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);
CREATE INDEX idx_contributor_site_id ON contributor(site_id);
CREATE UNIQUE INDEX idx_contributor_site_handle ON contributor(site_id, handle);

-- +migrate Down
DROP TABLE contributor;
