PRAGMA foreign_keys= OFF;
BEGIN;

ALTER TABLE oidc_clients
    ADD COLUMN client_type TEXT NOT NULL DEFAULT 'standard';
ALTER TABLE oidc_clients
    ADD COLUMN metadata_expires_at DATETIME;
ALTER TABLE oidc_clients
    ADD COLUMN metadata_grant_types TEXT;

COMMIT;
PRAGMA foreign_keys= ON;
