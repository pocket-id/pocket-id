PRAGMA foreign_keys=OFF;
BEGIN;
ALTER TABLE users ADD COLUMN ldap_id TEXT;
ALTER TABLE user_groups ADD COLUMN ldap_id TEXT;

CREATE UNIQUE INDEX users_ldap_id ON users (ldap_id);
CREATE UNIQUE INDEX user_groups_ldap_id ON user_groups (ldap_id);
COMMIT;
PRAGMA foreign_keys=ON;
