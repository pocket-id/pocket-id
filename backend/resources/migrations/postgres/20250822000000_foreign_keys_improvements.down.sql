ALTER TABLE audit_logs
    DROP CONSTRAINT IF EXISTS audit_logs_user_id_fkey,
    ADD CONSTRAINT audit_logs_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users (id);

ALTER TABLE oidc_authorization_codes
    DROP CONSTRAINT IF EXISTS oidc_authorization_codes_client_fk;