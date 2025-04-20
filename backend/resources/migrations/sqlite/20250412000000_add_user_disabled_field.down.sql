DROP INDEX idx_users_disabled;

ALTER TABLE users
DROP COLUMN disabled;