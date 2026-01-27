-- +migrate Up
ALTER TABLE user ADD COLUMN role TEXT NOT NULL DEFAULT 'editor';

-- +migrate Down
ALTER TABLE user DROP COLUMN role;
