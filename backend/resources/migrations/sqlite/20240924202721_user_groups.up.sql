PRAGMA foreign_keys=OFF;
BEGIN;
CREATE TABLE user_groups
(
    id           TEXT NOT NULL PRIMARY KEY,
    created_at   DATETIME,
    friendly_name TEXT NOT NULL,
    name        TEXT NOT NULL UNIQUE
);

CREATE TABLE user_groups_users
(
    user_id  TEXT NOT NULL,
    user_group_id TEXT NOT NULL,
    PRIMARY KEY (user_id, user_group_id),
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    FOREIGN KEY (user_group_id) REFERENCES user_groups (id) ON DELETE CASCADE
);
COMMIT;
PRAGMA foreign_keys=ON;
