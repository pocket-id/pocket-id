CREATE TABLE apis (
    id UUID NOT NULL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ,
    name TEXT NOT NULL,
    audience TEXT NOT NULL UNIQUE
);

CREATE TABLE api_permissions (
    id UUID NOT NULL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL,
    api_id UUID NOT NULL REFERENCES apis(id) ON DELETE CASCADE,
    key TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    UNIQUE (api_id, key)
);

CREATE INDEX idx_api_permissions_api_id ON api_permissions(api_id);

CREATE TABLE oidc_clients_allowed_api_permissions (
    oidc_client_id TEXT NOT NULL REFERENCES oidc_clients(id) ON DELETE CASCADE,
    api_permission_id UUID NOT NULL REFERENCES api_permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (oidc_client_id, api_permission_id)
);
