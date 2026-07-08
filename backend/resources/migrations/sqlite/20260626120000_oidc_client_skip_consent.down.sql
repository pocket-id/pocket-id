PRAGMA foreign_keys=OFF;
BEGIN;
ALTER TABLE oidc_clients DROP COLUMN skip_consent;
COMMIT;
PRAGMA foreign_keys=ON;
