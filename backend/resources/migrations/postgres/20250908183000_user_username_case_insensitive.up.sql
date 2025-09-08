BEGIN;

-- Normalize all existing usernames to lowercase to enforce case-insensitive semantics
UPDATE users
SET username = lower(username);

-- Enforce case-insensitive uniqueness at the DB layer using a functional unique index
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_indexes
        WHERE schemaname = 'public'
          AND indexname = 'users_username_lower_unique'
    ) THEN
        CREATE UNIQUE INDEX users_username_lower_unique
            ON public.users (lower(username));
    END IF;
END $$;

COMMIT;