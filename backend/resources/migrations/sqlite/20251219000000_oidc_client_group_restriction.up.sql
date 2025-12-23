PRAGMA foreign_keys= OFF;
BEGIN;

ALTER TABLE oidc_clients
    ADD COLUMN is_group_restricted BOOLEAN NOT NULL DEFAULT 0;

UPDATE oidc_clients
SET is_group_restricted = (SELECT CASE WHEN COUNT(*) > 0 THEN 1 ELSE 0 END
                           FROM oidc_clients_allowed_user_groups
                           WHERE oidc_clients_allowed_user_groups.oidc_client_id = oidc_clients.id);

COMMIT;
PRAGMA foreign_keys= ON;