PRAGMA foreign_keys=OFF;
BEGIN;
ALTER TABLE oidc_clients ADD COLUMN dark_image_type TEXT;
COMMIT;
PRAGMA foreign_keys=ON;
