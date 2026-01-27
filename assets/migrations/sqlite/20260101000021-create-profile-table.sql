-- +migrate Up
CREATE TABLE profile (
    id TEXT PRIMARY KEY,
    short_id TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
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
CREATE UNIQUE INDEX idx_profile_slug ON profile(slug);

-- +migrate Down
DROP TABLE profile;
