PRAGMA foreign_keys=OFF;
BEGIN;
UPDATE user_groups SET ldap_id = null WHERE ldap_id = '';
COMMIT;
PRAGMA foreign_keys=ON;
