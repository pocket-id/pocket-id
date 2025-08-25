PRAGMA foreign_keys=OFF;
BEGIN;
DELETE FROM app_config_variables WHERE value = '';
COMMIT;
PRAGMA foreign_keys=ON;
