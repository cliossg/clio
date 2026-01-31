-- +migrate Up
CREATE TABLE IF NOT EXISTS content (
    id TEXT PRIMARY KEY,
    site_id TEXT NOT NULL,
    user_id TEXT,
    short_id TEXT,
    section_id TEXT,
    kind TEXT,
    heading TEXT NOT NULL,
    summary TEXT,
    body TEXT,
    draft INTEGER DEFAULT 0,
    featured INTEGER DEFAULT 0,
    series TEXT,
    series_order INTEGER,
    published_at TIMESTAMP,
    created_by TEXT,
    updated_by TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    contributor_id TEXT,
    contributor_handle TEXT NOT NULL DEFAULT '',
    author_username TEXT NOT NULL DEFAULT '',
    hero_title_dark INTEGER DEFAULT 0,
    images_meta TEXT,
    FOREIGN KEY (site_id) REFERENCES site(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES user(id),
    FOREIGN KEY (section_id) REFERENCES section(id),
    FOREIGN KEY (contributor_id) REFERENCES contributor(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_content_site_id ON content(site_id);
CREATE INDEX IF NOT EXISTS idx_content_section_id ON content(section_id);
CREATE INDEX IF NOT EXISTS idx_content_user_id ON content(user_id);
CREATE INDEX IF NOT EXISTS idx_content_draft ON content(draft);
CREATE INDEX IF NOT EXISTS idx_content_published_at ON content(published_at);

-- +migrate Down
DROP TABLE IF EXISTS content;
