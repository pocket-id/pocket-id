DROP INDEX idx_api_keys_previous_key;

ALTER TABLE api_keys
    DROP COLUMN previous_key,
    DROP COLUMN previous_key_expires_at,
    DROP COLUMN scopes;
