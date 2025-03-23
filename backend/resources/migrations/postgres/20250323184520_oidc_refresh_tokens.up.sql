CREATE TABLE oidc_refresh_tokens (
    id UUID NOT NULL PRIMARY KEY,
    created_at TIMESTAMPTZ,
    token VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    scope TEXT NOT NULL,
    user_id UUID NOT NULL REFERENCES users ON DELETE CASCADE,
    client_id UUID NOT NULL REFERENCES oidc_clients ON DELETE CASCADE
);

CREATE INDEX idx_oidc_refresh_tokens_token ON oidc_refresh_tokens(token);