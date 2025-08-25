PRAGMA foreign_keys=OFF;
BEGIN;
ALTER TABLE oidc_clients ADD COLUMN pkce_enabled BOOLEAN DEFAULT FALSE;
COMMIT;
PRAGMA foreign_keys=ON;
