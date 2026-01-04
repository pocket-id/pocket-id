PRAGMA foreign_keys=OFF;
BEGIN;
ALTER TABLE oidc_device_codes DROP COLUMN nonce;
COMMIT;
PRAGMA foreign_keys=ON;
