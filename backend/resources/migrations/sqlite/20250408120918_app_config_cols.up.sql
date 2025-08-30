PRAGMA foreign_keys=OFF;
BEGIN;
ALTER TABLE app_config_variables DROP type;
ALTER TABLE app_config_variables DROP is_public;
ALTER TABLE app_config_variables DROP is_internal;
ALTER TABLE app_config_variables DROP default_value;
COMMIT;
PRAGMA foreign_keys=ON;
