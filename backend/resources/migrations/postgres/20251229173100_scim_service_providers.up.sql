CREATE TABLE scim_service_providers
(
    id             UUID PRIMARY KEY,
    created_at     TIMESTAMPTZ NOT NULL,
    endpoint       TEXT        NOT NULL,
    token          TEXT        NOT NULL,
    last_synced_at TIMESTAMPTZ,
    oidc_client_id UUID        NOT NULL REFERENCES oidc_clients (id) ON DELETE CASCADE
);

ALTER TABLE users
    ADD COLUMN updated_at TIMESTAMPTZ;

ALTER TABLE user_groups
    ADD COLUMN updated_at TIMESTAMPTZ;
