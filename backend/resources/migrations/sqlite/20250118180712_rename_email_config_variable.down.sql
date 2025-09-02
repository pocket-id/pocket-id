PRAGMA foreign_keys=OFF;
BEGIN;
UPDATE app_config_variables SET key = 'emailEnabled' WHERE key = 'emailLoginNotificationEnabled';
COMMIT;
PRAGMA foreign_keys=ON;
