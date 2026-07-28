PRAGMA foreign_keys=OFF;
BEGIN;

-- One-time access tokens are now stored in the actor state store, so the table is no longer needed.
DROP TABLE IF EXISTS one_time_access_tokens;

-- Freeze the signup tokens.
-- Encode every signup token (with its user group IDs) as a single JSON array and store it in the "kv" table under the "signup_tokens_migrated" key, so the singleton signup token actor can seed its state from it on first startup.
-- The "HAVING count(*) > 0" clause ensures nothing is written to the "kv" table when there are no signup tokens.
-- Timestamps are stored as Unix seconds, matching how DateTime values are persisted on SQLite.
INSERT INTO kv ("key", "value")
SELECT 'signup_tokens_migrated', json_group_array(
    json_object(
        'id', st.id,
        'token', st.token,
        'expiresAt', st.expires_at,
        'usageLimit', st.usage_limit,
        'usageCount', st.usage_count,
        'createdAt', st.created_at,
        'userGroupIds', json((
            SELECT COALESCE(json_group_array(stug.user_group_id), json_array())
            FROM signup_tokens_user_groups stug
            WHERE stug.signup_token_id = st.id
        ))
    )
)
FROM signup_tokens st
HAVING count(*) > 0;

-- Drop the now-frozen signup token tables.
DROP TABLE signup_tokens_user_groups;
DROP TABLE signup_tokens;

COMMIT;
PRAGMA foreign_keys=ON;
