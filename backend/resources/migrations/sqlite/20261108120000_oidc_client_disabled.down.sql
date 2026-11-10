PRAGMA foreign_keys=OFF;
BEGIN;
ALTER TABLE oidc_clients DROP COLUMN disabled;
COMMIT;
PRAGMA foreign_keys=ON;
