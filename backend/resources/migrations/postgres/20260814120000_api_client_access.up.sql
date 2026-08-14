ALTER TABLE apis ADD COLUMN allow_cimd_clients BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE api_permissions ADD COLUMN allowed_for_cimd_clients BOOLEAN NOT NULL DEFAULT FALSE;

-- Access to an API is tracked separately from the permissions granted for it, because a client may be allowed to request tokens for an API without any scope
CREATE TABLE oidc_clients_allowed_apis (
    oidc_client_id TEXT NOT NULL REFERENCES oidc_clients(id) ON DELETE CASCADE,
    api_id UUID NOT NULL REFERENCES apis(id) ON DELETE CASCADE,
    subject_type TEXT NOT NULL CHECK (subject_type IN ('user', 'client')),
    PRIMARY KEY (oidc_client_id, api_id, subject_type)
);

-- The primary key leads with oidc_client_id, so the API-side lookups and the cascade from apis need their own index
CREATE INDEX idx_oidc_clients_allowed_apis_api_id ON oidc_clients_allowed_apis(api_id);

-- Every existing permission grant implied access to its API, so those clients keep exactly the access they had
INSERT INTO oidc_clients_allowed_apis (oidc_client_id, api_id, subject_type)
SELECT DISTINCT g.oidc_client_id, p.api_id, g.subject_type
FROM oidc_clients_allowed_api_permissions g
JOIN api_permissions p ON p.id = g.api_permission_id;
