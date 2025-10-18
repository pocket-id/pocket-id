PRAGMA foreign_keys=OFF;
BEGIN;
ALTER TABLE oidc_clients DROP COLUMN dark_image_type;
COMMIT;
PRAGMA foreign_keys=ON;
