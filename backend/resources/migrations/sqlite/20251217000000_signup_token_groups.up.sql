PRAGMA foreign_keys=OFF;
BEGIN;

CREATE TABLE signup_tokens_user_groups
(
    signup_token_id TEXT NOT NULL,
    user_group_id   TEXT NOT NULL,
    PRIMARY KEY (signup_token_id, user_group_id),
    FOREIGN KEY (signup_token_id) REFERENCES signup_tokens (id) ON DELETE CASCADE,
    FOREIGN KEY (user_group_id) REFERENCES user_groups (id) ON DELETE CASCADE
);

COMMIT;
PRAGMA foreign_keys=ON;