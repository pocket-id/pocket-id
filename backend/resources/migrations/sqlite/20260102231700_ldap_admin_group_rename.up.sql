PRAGMA foreign_keys= OFF;
BEGIN;

UPDATE app_config_variables
SET key = 'ldapAdminGroupName'
WHERE key = 'ldapAttributeAdminGroup'
  AND NOT EXISTS (
    SELECT 1
    FROM app_config_variables
    WHERE key = 'ldapAdminGroupName'
);

COMMIT;
PRAGMA foreign_keys= ON;
