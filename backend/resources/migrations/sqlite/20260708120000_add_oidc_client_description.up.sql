PRAGMA foreign_keys=OFF;
BEGIN;

ALTER TABLE oidc_clients ADD COLUMN description TEXT NOT NULL DEFAULT '';

COMMIT;
PRAGMA foreign_keys=ON;
