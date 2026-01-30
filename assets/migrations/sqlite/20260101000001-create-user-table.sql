-- +migrate Up
CREATE TABLE IF NOT EXISTS user (
    id TEXT PRIMARY KEY,
    short_id TEXT NOT NULL DEFAULT '',
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    must_change_password INTEGER NOT NULL DEFAULT 0,
    roles TEXT NOT NULL DEFAULT 'editor',
    profile_id TEXT
);

CREATE INDEX IF NOT EXISTS idx_user_email ON user(email);
CREATE INDEX IF NOT EXISTS idx_user_status ON user(status);

-- +migrate Down
DROP TABLE IF EXISTS user;
