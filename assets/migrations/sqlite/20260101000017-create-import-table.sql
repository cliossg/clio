-- +migrate Up
CREATE TABLE IF NOT EXISTS import (
    id TEXT PRIMARY KEY,
    short_id TEXT NOT NULL,
    file_path TEXT NOT NULL,
    file_hash TEXT,
    file_mtime TIMESTAMP,
    content_id TEXT,
    site_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    imported_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (content_id) REFERENCES content(id) ON DELETE SET NULL,
    FOREIGN KEY (site_id) REFERENCES site(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES user(id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_import_site_id ON import(site_id);
CREATE INDEX IF NOT EXISTS idx_import_content_id ON import(content_id);
CREATE INDEX IF NOT EXISTS idx_import_status ON import(status);
CREATE UNIQUE INDEX IF NOT EXISTS idx_import_site_file_path ON import(site_id, file_path);

-- +migrate Down
DROP TABLE IF EXISTS import;
