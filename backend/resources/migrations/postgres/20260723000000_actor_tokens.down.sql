-- Recreate the one_time_access_tokens table with the schema it had before it was dropped.
CREATE TABLE one_time_access_tokens
(
    id           UUID NOT NULL PRIMARY KEY,
    created_at   TIMESTAMPTZ,
    token        VARCHAR(255) NOT NULL UNIQUE,
    expires_at   TIMESTAMPTZ NOT NULL,
    user_id      UUID NOT NULL REFERENCES users ON DELETE CASCADE,
    device_token VARCHAR(16)
);
CREATE INDEX IF NOT EXISTS idx_one_time_access_tokens_expires_at ON one_time_access_tokens (expires_at);

-- Recreate the signup token tables with the schema they had before they were frozen.
CREATE TABLE signup_tokens (
    id UUID NOT NULL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL,
    token VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    usage_limit INTEGER NOT NULL DEFAULT 1,
    usage_count INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_signup_tokens_token ON signup_tokens(token);
CREATE INDEX idx_signup_tokens_expires_at ON signup_tokens(expires_at);

CREATE TABLE signup_tokens_user_groups
(
    signup_token_id UUID NOT NULL,
    user_group_id   UUID NOT NULL,
    PRIMARY KEY (signup_token_id, user_group_id),
    FOREIGN KEY (signup_token_id) REFERENCES signup_tokens (id) ON DELETE CASCADE,
    FOREIGN KEY (user_group_id) REFERENCES user_groups (id) ON DELETE CASCADE
);

-- Restore the signup tokens from the frozen JSON document stored in the "kv" table.
-- json_array_elements expands the JSON array into one row per token object.
INSERT INTO signup_tokens (id, created_at, token, expires_at, usage_limit, usage_count)
SELECT
    (e ->> 'id')::uuid,
    to_timestamp((e ->> 'createdAt')::bigint),
    e ->> 'token',
    to_timestamp((e ->> 'expiresAt')::bigint),
    (e ->> 'usageLimit')::int,
    (e ->> 'usageCount')::int
FROM kv, json_array_elements(kv."value"::json) AS e
WHERE kv."key" = 'signup_tokens_migrated';

-- Restore the token/user-group associations, expanding each token's nested userGroupIds array.
INSERT INTO signup_tokens_user_groups (signup_token_id, user_group_id)
SELECT
    (e ->> 'id')::uuid,
    g.value::uuid
FROM kv, json_array_elements(kv."value"::json) AS e, json_array_elements_text(e -> 'userGroupIds') AS g
WHERE kv."key" = 'signup_tokens_migrated';

-- Remove the frozen signup tokens from the "kv" table.
DELETE FROM kv WHERE "key" = 'signup_tokens_migrated';
