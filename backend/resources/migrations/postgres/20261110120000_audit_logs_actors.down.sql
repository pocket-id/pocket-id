DROP INDEX IF EXISTS idx_audit_logs_actor_id;

ALTER TABLE audit_logs
    DROP COLUMN actor_type,
    DROP COLUMN actor_id,
    DROP COLUMN actor_name,
    DROP CONSTRAINT IF EXISTS audit_logs_user_id_fkey,
    ADD CONSTRAINT audit_logs_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE;
