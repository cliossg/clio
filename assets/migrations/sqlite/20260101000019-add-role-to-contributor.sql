-- +migrate Up
ALTER TABLE contributor ADD COLUMN role TEXT NOT NULL DEFAULT 'editor';

-- +migrate Down
ALTER TABLE contributor DROP COLUMN role;
