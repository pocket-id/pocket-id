PRAGMA foreign_keys=OFF;
BEGIN;

ALTER TABLE oidc_clients DROP COLUMN is_group_restricted;

COMMIT;
PRAGMA foreign_keys=ON;