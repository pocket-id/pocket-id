CREATE TABLE oidc_reauthentication_tokens (
    id TEXT PRIMARY KEY,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    token TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    used BOOLEAN NOT NULL DEFAULT FALSE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    client_id TEXT NOT NULL REFERENCES oidc_clients(id) ON DELETE CASCADE
);

CREATE INDEX idx_oidc_reauthentication_tokens_token ON oidc_reauthentication_tokens(token);
CREATE INDEX idx_oidc_reauthentication_tokens_user_id ON oidc_reauthentication_tokens(user_id);
CREATE INDEX idx_oidc_reauthentication_tokens_client_id ON oidc_reauthentication_tokens(client_id);
CREATE INDEX idx_oidc_reauthentication_tokens_expires_at ON oidc_reauthentication_tokens(expires_at);