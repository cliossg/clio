-- +migrate Up
ALTER TABLE content ADD COLUMN contributor_id TEXT REFERENCES contributor(id) ON DELETE SET NULL;

-- +migrate Down
ALTER TABLE content DROP COLUMN contributor_id;
