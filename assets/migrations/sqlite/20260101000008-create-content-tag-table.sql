-- +migrate Up
CREATE TABLE IF NOT EXISTS content_tag (
    id TEXT PRIMARY KEY,
    content_id TEXT NOT NULL,
    tag_id TEXT NOT NULL,
    created_at TIMESTAMP,
    FOREIGN KEY (content_id) REFERENCES content(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tag(id) ON DELETE CASCADE,
    UNIQUE(content_id, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_content_tag_content_id ON content_tag(content_id);
CREATE INDEX IF NOT EXISTS idx_content_tag_tag_id ON content_tag(tag_id);

-- +migrate Down
DROP TABLE IF EXISTS content_tag;
