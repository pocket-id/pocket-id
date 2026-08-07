PRAGMA foreign_keys= OFF;
BEGIN;

ALTER TABLE oidc_clients
    DROP COLUMN registration_access_token_hash;

COMMIT;
PRAGMA foreign_keys= ON;
