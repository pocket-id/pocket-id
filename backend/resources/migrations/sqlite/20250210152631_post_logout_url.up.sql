PRAGMA foreign_keys=OFF;
BEGIN;
ALTER TABLE oidc_clients ADD COLUMN logout_callback_urls BLOB;
COMMIT;
PRAGMA foreign_keys=ON;
