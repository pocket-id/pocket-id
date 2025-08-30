PRAGMA foreign_keys=OFF;
BEGIN;
ALTER TABLE app_config_variables DROP COLUMN default_value;
COMMIT;
PRAGMA foreign_keys=ON;
