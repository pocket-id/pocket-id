ALTER TABLE users ADD COLUMN display_name TEXT;

UPDATE users
SET display_name = trim(coalesce(first_name,'') || ' ' || coalesce(last_name,''));

ALTER TABLE users ALTER COLUMN display_name SET NOT NULL;