BEGIN;

DROP INDEX idx_api_keys_previous_key;

ALTER TABLE api_keys DROP COLUMN previous_key;
ALTER TABLE api_keys DROP COLUMN previous_key_expires_at;
ALTER TABLE api_keys DROP COLUMN scopes;

COMMIT;
