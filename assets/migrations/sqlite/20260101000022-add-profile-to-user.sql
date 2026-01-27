-- +migrate Up
ALTER TABLE user ADD COLUMN profile_id TEXT REFERENCES profile(id) ON DELETE SET NULL;

-- +migrate Down
ALTER TABLE user DROP COLUMN profile_id;
