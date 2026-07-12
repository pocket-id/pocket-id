ALTER TABLE audit_logs
    DROP CONSTRAINT IF EXISTS audit_logs_user_id_fkey,
    ADD CONSTRAINT audit_logs_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE;

ALTER TABLE oidc_authorization_codes
    ADD CONSTRAINT oidc_authorization_codes_client_fk
        FOREIGN KEY (client_id) REFERENCES oidc_clients (id) ON DELETE CASCADE;