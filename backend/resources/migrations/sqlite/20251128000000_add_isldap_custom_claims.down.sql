PRAGMA foreign_keys=OFF;
BEGIN;
ALTER TABLE custom_claims RENAME TO custom_claims_old;
CREATE TABLE custom_claims
(
    id            TEXT NOT NULL PRIMARY KEY,
    created_at    DATETIME,
    key           TEXT NOT NULL,
    value         TEXT NOT NULL,

    user_id       TEXT,
    user_group_id TEXT,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    FOREIGN KEY (user_group_id) REFERENCES user_groups (id) ON DELETE CASCADE,

    CONSTRAINT custom_claims_unique UNIQUE (key, user_id, user_group_id),
    CHECK (user_id IS NOT NULL OR user_group_id IS NOT NULL)
);
INSERT INTO custom_claims (id, created_at, key, value, user_id, user_group_id)
SELECT id, created_at, key, value, user_id, user_group_id, false FROM custom_claims_old
WHERE is_ldap = 0;
DROP TABLE custom_claims_old;
delete from app_config_variables where key = 'ldapExtraAttributes';
COMMIT;
PRAGMA foreign_keys=ON;
