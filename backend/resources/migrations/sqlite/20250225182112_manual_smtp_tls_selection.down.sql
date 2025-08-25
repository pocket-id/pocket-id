PRAGMA foreign_keys=OFF;
BEGIN;
UPDATE app_config_variables SET value = 'true' WHERE key = 'smtpTls';
COMMIT;
PRAGMA foreign_keys=ON;
