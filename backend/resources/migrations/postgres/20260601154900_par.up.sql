CREATE TABLE oidc_pushed_authorization_requests (
    id UUID NOT NULL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL,
    request_uri TEXT NOT NULL UNIQUE,
    client_id TEXT NOT NULL REFERENCES oidc_clients(id) ON DELETE CASCADE,
    parameters JSONB NOT NULL DEFAULT '{}',
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_oidc_par_expires_at ON oidc_pushed_authorization_requests (expires_at);

ALTER TABLE oidc_clients ADD COLUMN requires_pushed_authorization_requests BOOLEAN NOT NULL DEFAULT FALSE;
