PRAGMA foreign_keys=OFF;
BEGIN;
ALTER TABLE oidc_clients DROP COLUMN logout_callback_urls;
COMMIT;
PRAGMA foreign_keys=ON;
