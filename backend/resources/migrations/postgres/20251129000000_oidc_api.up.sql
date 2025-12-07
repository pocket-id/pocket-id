CREATE TABLE oidc_apis
(
    id UUID PRIMARY KEY NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    name TEXT NOT NULL,
    identifier TEXT NOT NULL,
    data JSONB NOT NULL
);

CREATE UNIQUE INDEX idx_oidc_apis_identifier_key ON oidc_apis(identifier);
CREATE INDEX idx_oidc_apis_name_key ON oidc_apis(name);
