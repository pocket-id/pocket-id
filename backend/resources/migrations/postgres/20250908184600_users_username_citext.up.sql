BEGIN;

-- Ensure case-insensitive semantics at the column level using CITEXT
CREATE EXTENSION IF NOT EXISTS citext;

-- Sanitize existing usernames to ASCII-only allowed charset and lowercase
-- This prevents conflicts when switching to case-insensitive semantics
UPDATE users
SET username = lower(regexp_replace(username, '[^a-z0-9_.@-]', '', 'gi'));

-- Drop old functional unique index if present (from previous approach)
DROP INDEX IF EXISTS users_username_lower_unique;

-- Convert column to CITEXT (case-insensitive)
ALTER TABLE users
    ALTER COLUMN username TYPE CITEXT
    USING username::citext;

-- Ensure a unique constraint/index exists on username under case-insensitive semantics
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_indexes
        WHERE schemaname = 'public'
          AND indexname = 'users_username_unique'
    ) THEN
        CREATE UNIQUE INDEX users_username_unique ON public.users (username);
    END IF;
END $$;

COMMIT;