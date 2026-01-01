PRAGMA foreign_keys=OFF;
BEGIN;
ALTER TABLE oidc_device_codes ADD COLUMN nonce VARCHAR(255);
COMMIT;
PRAGMA foreign_keys=ON;
