CREATE TABLE oidc_refresh_tokens (
    id TEXT NOT NULL PRIMARY KEY,
    created_at DATETIME,
    token TEXT NOT NULL UNIQUE,
    expires_at DATETIME NOT NULL,
    scope TEXT NOT NULL,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    client_id TEXT NOT NULL REFERENCES oidc_clients(id) ON DELETE CASCADE
);

CREATE INDEX idx_oidc_refresh_tokens_token ON oidc_refresh_tokens(token);