ALTER TABLE users
ADD COLUMN disabled BOOLEAN DEFAULT FALSE NOT NULL;

CREATE INDEX idx_users_disabled ON users (disabled);