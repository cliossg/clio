-- +migrate Up
ALTER TABLE user RENAME COLUMN role TO roles;

-- +migrate Down
ALTER TABLE user RENAME COLUMN roles TO role;
