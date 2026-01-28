-- +migrate Up
ALTER TABLE param ADD COLUMN category TEXT DEFAULT '';
ALTER TABLE param ADD COLUMN position INTEGER DEFAULT 0;

-- +migrate Down
ALTER TABLE param DROP COLUMN category;
ALTER TABLE param DROP COLUMN position;
