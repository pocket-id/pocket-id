PRAGMA foreign_keys= OFF;
BEGIN;

ALTER TABLE oidc_clients
    ADD COLUMN client_type TEXT NOT NULL DEFAULT 'standard';
ALTER TABLE oidc_clients
    ADD COLUMN metadata_expires_at DATETIME;

COMMIT;
PRAGMA foreign_keys= ON;
