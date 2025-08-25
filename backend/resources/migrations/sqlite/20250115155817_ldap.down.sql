PRAGMA foreign_keys=OFF;
BEGIN;
ALTER TABLE users DROP COLUMN ldap_id;
ALTER TABLE user_groups DROP COLUMN ldap_id;
COMMIT;
PRAGMA foreign_keys=ON;
