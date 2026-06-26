ALTER TABLE oidc_clients
    ADD COLUMN is_metadata_document BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE oidc_clients
    ADD COLUMN metadata_expires_at TIMESTAMPTZ;
