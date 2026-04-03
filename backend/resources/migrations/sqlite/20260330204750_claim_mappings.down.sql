PRAGMA foreign_keys=OFF;
BEGIN;
ALTER TABLE oidc_clients DROP COLUMN claim_mappings;

COMMIT;
PRAGMA foreign_keys=ON;
