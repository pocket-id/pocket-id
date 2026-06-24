PRAGMA foreign_keys= OFF;
BEGIN;

ALTER TABLE oidc_clients
    ADD COLUMN is_metadata_document BOOLEAN NOT NULL DEFAULT 0;
ALTER TABLE oidc_clients
    ADD COLUMN metadata_expires_at DATETIME;

COMMIT;
PRAGMA foreign_keys= ON;
