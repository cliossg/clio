-- +migrate Up
ALTER TABLE site ADD COLUMN default_layout_id TEXT;
ALTER TABLE site ADD COLUMN default_layout_name TEXT;

-- +migrate Down
-- SQLite doesn't support DROP COLUMN, would need table recreation
