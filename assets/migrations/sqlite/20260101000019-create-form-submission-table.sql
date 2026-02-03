-- +migrate Up
CREATE TABLE IF NOT EXISTS form_submission (
    id TEXT PRIMARY KEY,
    site_id TEXT NOT NULL,
    form_type TEXT NOT NULL DEFAULT 'contact',
    name TEXT,
    email TEXT,
    message TEXT NOT NULL,
    ip_address TEXT,
    user_agent TEXT,
    read_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (site_id) REFERENCES site(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_form_submission_site_id ON form_submission(site_id);
CREATE INDEX IF NOT EXISTS idx_form_submission_created_at ON form_submission(created_at DESC);

-- +migrate Down
DROP TABLE IF EXISTS form_submission;
