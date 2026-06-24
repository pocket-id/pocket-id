PRAGMA foreign_keys=OFF;
BEGIN;

ALTER TABLE oidc_clients ADD COLUMN description TEXT;

COMMIT;
PRAGMA foreign_keys=ON;
