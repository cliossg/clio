-- +migrate Up
CREATE TABLE profile (
    id TEXT PRIMARY KEY,
    site_id TEXT NOT NULL REFERENCES site(id) ON DELETE CASCADE,
    short_id TEXT NOT NULL,
    slug TEXT NOT NULL,
    name TEXT NOT NULL,
    surname TEXT NOT NULL DEFAULT '',
    bio TEXT NOT NULL DEFAULT '',
    social_links TEXT NOT NULL DEFAULT '[]',
    photo_path TEXT NOT NULL DEFAULT '',
    created_by TEXT NOT NULL,
    updated_by TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);
CREATE UNIQUE INDEX idx_profile_site_slug ON profile(site_id, slug);

-- +migrate Down
DROP TABLE profile;
