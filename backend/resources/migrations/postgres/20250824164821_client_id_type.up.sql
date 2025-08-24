-- Drop foreign keys that reference oidc_clients(id)
ALTER TABLE oidc_authorization_codes
    DROP CONSTRAINT IF EXISTS oidc_authorization_codes_client_fk;
ALTER TABLE user_authorized_oidc_clients
    DROP CONSTRAINT IF EXISTS user_authorized_oidc_clients_client_id_fkey;
ALTER TABLE oidc_refresh_tokens
    DROP CONSTRAINT IF EXISTS oidc_refresh_tokens_client_id_fkey;
ALTER TABLE oidc_device_codes
    DROP CONSTRAINT IF EXISTS oidc_device_codes_client_id_fkey;
ALTER TABLE oidc_clients_allowed_user_groups
    DROP CONSTRAINT IF EXISTS oidc_clients_allowed_user_groups_oidc_client_id_fkey;

-- Alter child columns to TEXT
ALTER TABLE oidc_authorization_codes
    ALTER COLUMN client_id TYPE TEXT USING client_id::text;

ALTER TABLE user_authorized_oidc_clients
    ALTER
        COLUMN client_id TYPE TEXT USING client_id::text;

ALTER TABLE oidc_refresh_tokens
    ALTER
        COLUMN client_id TYPE TEXT USING client_id::text;

ALTER TABLE oidc_device_codes
    ALTER
        COLUMN client_id TYPE TEXT USING client_id::text;

ALTER TABLE oidc_clients_allowed_user_groups
    ALTER
        COLUMN oidc_client_id TYPE TEXT USING oidc_client_id::text;

-- Alter parent primary key column to TEXT
ALTER TABLE oidc_clients
    ALTER
        COLUMN id TYPE TEXT USING id::text;

-- Recreate foreign keys with the new type
ALTER TABLE oidc_authorization_codes
    ADD CONSTRAINT oidc_authorization_codes_client_fk
        FOREIGN KEY (client_id) REFERENCES oidc_clients (id) ON DELETE CASCADE;

ALTER TABLE user_authorized_oidc_clients
    ADD CONSTRAINT user_authorized_oidc_clients_client_id_fkey
        FOREIGN KEY (client_id) REFERENCES oidc_clients (id) ON DELETE CASCADE;

ALTER TABLE oidc_refresh_tokens
    ADD CONSTRAINT oidc_refresh_tokens_client_id_fkey
        FOREIGN KEY (client_id) REFERENCES oidc_clients (id) ON DELETE CASCADE;

ALTER TABLE oidc_device_codes
    ADD CONSTRAINT oidc_device_codes_client_id_fkey
        FOREIGN KEY (client_id) REFERENCES oidc_clients (id) ON DELETE CASCADE;

ALTER TABLE oidc_clients_allowed_user_groups
    ADD CONSTRAINT oidc_clients_allowed_user_groups_oidc_client_id_fkey
        FOREIGN KEY (oidc_client_id) REFERENCES oidc_clients (id) ON DELETE CASCADE;