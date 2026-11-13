-- The static API key now acts as itself instead of as a synthetic admin user
UPDATE audit_logs
SET actor_type = 'static_api_key',
    actor_id   = NULL,
    actor_name = NULL
WHERE actor_type = 'user'
  AND actor_id = '00000000-0000-0000-0000-000000000000';

DELETE FROM users WHERE id = '00000000-0000-0000-0000-000000000000';
