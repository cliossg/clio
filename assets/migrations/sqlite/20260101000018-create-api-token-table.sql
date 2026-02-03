-- +migrate Up
CREATE TABLE IF NOT EXISTS api_token (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    token_hash TEXT NOT NULL,
    last_used_at TIMESTAMP,
    expires_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES user(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_api_token_user_id ON api_token(user_id);
CREATE INDEX IF NOT EXISTS idx_api_token_hash ON api_token(token_hash);

-- +migrate Down
DROP TABLE IF EXISTS api_token;
