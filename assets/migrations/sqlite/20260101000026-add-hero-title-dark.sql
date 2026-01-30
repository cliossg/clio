-- +migrate Up
ALTER TABLE content ADD COLUMN hero_title_dark INTEGER DEFAULT 0;

-- +migrate Down
-- SQLite doesn't support DROP COLUMN directly, so we leave it
