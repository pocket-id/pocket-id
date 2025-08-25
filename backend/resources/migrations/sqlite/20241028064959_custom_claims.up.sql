PRAGMA foreign_keys=OFF;
BEGIN;
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
COMMIT;
PRAGMA foreign_keys=ON;
