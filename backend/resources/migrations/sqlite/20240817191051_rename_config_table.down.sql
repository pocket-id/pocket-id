PRAGMA foreign_keys=OFF;
BEGIN;
ALTER TABLE app_config_variables
    RENAME TO application_configuration_variables;
COMMIT;
PRAGMA foreign_keys=ON;
