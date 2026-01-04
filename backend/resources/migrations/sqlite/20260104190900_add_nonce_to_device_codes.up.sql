PRAGMA foreign_keys=OFF;
BEGIN;
ALTER TABLE oidc_device_codes ADD COLUMN nonce TEXT;
COMMIT;
PRAGMA foreign_keys=ON;
