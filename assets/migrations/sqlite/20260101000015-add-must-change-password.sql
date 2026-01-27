-- +migrate Up
ALTER TABLE user ADD COLUMN must_change_password INTEGER NOT NULL DEFAULT 0;

-- +migrate Down
ALTER TABLE user DROP COLUMN must_change_password;
