PRAGMA foreign_keys=OFF;
BEGIN;
ALTER TABLE application_configuration_variables
    RENAME TO app_config_variables;
COMMIT;
PRAGMA foreign_keys=ON;
