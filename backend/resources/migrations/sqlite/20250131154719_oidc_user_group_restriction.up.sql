PRAGMA foreign_keys=OFF;
BEGIN;
CREATE TABLE oidc_clients_allowed_user_groups
(
    user_group_id  TEXT NOT NULL,
    oidc_client_id TEXT NOT NULL,
    PRIMARY KEY (oidc_client_id, user_group_id),
    FOREIGN KEY (oidc_client_id) REFERENCES oidc_clients (id) ON DELETE CASCADE,
    FOREIGN KEY (user_group_id) REFERENCES user_groups (id) ON DELETE CASCADE
);
COMMIT;
PRAGMA foreign_keys=ON;
