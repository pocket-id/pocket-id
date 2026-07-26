-- One-time access tokens are now stored in the actor state store, so the table is no longer needed.
DROP TABLE IF EXISTS one_time_access_tokens;

-- Freeze the signup tokens.
-- Encode every signup token (with its user group IDs) as a single JSON array and store it in the "kv" table under the "signup_tokens_migrated" key, so the singleton signup token actor can seed its state from it on first startup.
-- The "HAVING count(*) > 0" clause ensures nothing is written to the "kv" table when there are no signup tokens.
-- Timestamps are stored as Unix seconds so the frozen format is identical across databases.
INSERT INTO kv ("key", "value")
SELECT 'signup_tokens_migrated', json_agg(
    json_build_object(
        'id', st.id,
        'token', st.token,
        'expiresAt', extract(epoch FROM st.expires_at)::bigint,
        'usageLimit', st.usage_limit,
        'usageCount', st.usage_count,
        'createdAt', extract(epoch FROM st.created_at)::bigint,
        'userGroupIds', COALESCE(
            (SELECT json_agg(stug.user_group_id) FROM signup_tokens_user_groups stug WHERE stug.signup_token_id = st.id),
            '[]'::json
        )
    )
)::text
FROM signup_tokens st
HAVING count(*) > 0;

-- Drop the now-frozen signup token tables.
DROP TABLE signup_tokens_user_groups;
DROP TABLE signup_tokens;
