PRAGMA foreign_keys=OFF;
BEGIN;

ALTER TABLE oidc_clients DROP COLUMN pkce_supported;

COMMIT;
PRAGMA foreign_keys=ON;