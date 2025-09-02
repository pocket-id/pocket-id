PRAGMA foreign_keys=OFF;
BEGIN;
ALTER TABLE oidc_clients DROP COLUMN launch_url;

ALTER TABLE user_authorized_oidc_clients DROP COLUMN created_at;
COMMIT;
PRAGMA foreign_keys=ON;
