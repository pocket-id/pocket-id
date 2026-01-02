PRAGMA foreign_keys= OFF;
BEGIN;

UPDATE app_config_variables SET value = 'ldapAttributeAdminGroup' WHERE value = 'ldapAdminGroupName';

COMMIT;
PRAGMA foreign_keys= ON;
