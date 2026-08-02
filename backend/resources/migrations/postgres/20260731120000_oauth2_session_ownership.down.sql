DROP INDEX IF EXISTS idx_oauth2_sessions_client_subject;

ALTER TABLE oauth2_sessions
    DROP CONSTRAINT IF EXISTS chk_oauth2_sessions_client_id,
    DROP CONSTRAINT IF EXISTS fk_oauth2_sessions_client_id,
    DROP COLUMN IF EXISTS client_id;
