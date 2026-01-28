-- +migrate Up
ALTER TABLE param RENAME TO setting;

-- +migrate Down
ALTER TABLE setting RENAME TO param;
