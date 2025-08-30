PRAGMA foreign_keys=OFF;
BEGIN;
UPDATE users SET ldap_id = '' WHERE ldap_id IS NULL;
UPDATE user_groups SET ldap_id = '' WHERE ldap_id IS NULL;
COMMIT;
PRAGMA foreign_keys=ON;
