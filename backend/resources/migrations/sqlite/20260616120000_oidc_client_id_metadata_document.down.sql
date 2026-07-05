PRAGMA foreign_keys= OFF;
BEGIN;

ALTER TABLE oidc_clients DROP COLUMN client_type;
ALTER TABLE oidc_clients DROP COLUMN metadata_expires_at;

COMMIT;
PRAGMA foreign_keys= ON;
