-- +migrate Up
ALTER TABLE content ADD COLUMN contributor_id TEXT REFERENCES contributor(id) ON DELETE SET NULL;
ALTER TABLE content ADD COLUMN contributor_handle TEXT NOT NULL DEFAULT '';
ALTER TABLE content ADD COLUMN author_username TEXT NOT NULL DEFAULT '';

-- +migrate Down
ALTER TABLE content DROP COLUMN author_username;
ALTER TABLE content DROP COLUMN contributor_handle;
ALTER TABLE content DROP COLUMN contributor_id;
