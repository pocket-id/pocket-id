PRAGMA foreign_keys=OFF;
BEGIN;
ALTER TABLE oidc_clients ADD COLUMN credentials TEXT NULL;

COMMIT;
PRAGMA foreign_keys=ON;
