CREATE TABLE oidc_reauthentication_tokens (
    id TEXT PRIMARY KEY,
    created_at DATETIME,
    updated_at DATETIME,
    token TEXT NOT NULL UNIQUE,
    expires_at INTEGER NOT NULL,
    used INTEGER NOT NULL DEFAULT 0,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    client_id TEXT NOT NULL REFERENCES oidc_clients(id) ON DELETE CASCADE
);

CREATE INDEX idx_oidc_reauthentication_tokens_token ON oidc_reauthentication_tokens(token);