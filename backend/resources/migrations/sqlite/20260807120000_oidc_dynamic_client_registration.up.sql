PRAGMA foreign_keys= OFF;
BEGIN;

ALTER TABLE oidc_clients
    ADD COLUMN registration_access_token_hash TEXT;

COMMIT;
PRAGMA foreign_keys= ON;
