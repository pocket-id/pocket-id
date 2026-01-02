PRAGMA foreign_keys= OFF;
BEGIN;

UPDATE app_config_variables SET key = 'ldapAdminGroupName' WHERE key = 'ldapAttributeAdminGroup';

COMMIT;
PRAGMA foreign_keys= ON;
