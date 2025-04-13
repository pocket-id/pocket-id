ALTER TABLE users
ADD COLUMN disabled NUMERIC DEFAULT FALSE NOT NULL;

CREATE INDEX idx_users_disabled ON users (disabled);