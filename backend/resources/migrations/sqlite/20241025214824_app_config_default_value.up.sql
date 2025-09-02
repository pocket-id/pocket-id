PRAGMA foreign_keys=OFF;
BEGIN;
ALTER TABLE app_config_variables ADD COLUMN default_value TEXT;
COMMIT;
PRAGMA foreign_keys=ON;
