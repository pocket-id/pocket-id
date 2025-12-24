ALTER TABLE oidc_clients
    ADD COLUMN is_group_restricted boolean NOT NULL DEFAULT false;

UPDATE oidc_clients oc
SET is_group_restricted =
    EXISTS (
    SELECT 1
    FROM oidc_clients_allowed_user_groups a
    WHERE a.oidc_client_id = oc.id
    );