PRAGMA foreign_keys=OFF;
BEGIN;
UPDATE app_config_variables SET key = 'emailLoginNotificationEnabled' WHERE key = 'emailEnabled';
COMMIT;
PRAGMA foreign_keys=ON;
