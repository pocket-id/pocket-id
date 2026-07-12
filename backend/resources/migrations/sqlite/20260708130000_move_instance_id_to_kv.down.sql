PRAGMA foreign_keys=OFF;
BEGIN;

-- Move the instance ID back into the standalone app config table
INSERT INTO app_config_variables ("key", "value")
SELECT 'instanceId', "value"
    FROM kv
    WHERE "key" = 'instance_id' AND "value" IS NOT NULL
    ON CONFLICT ("key") DO NOTHING;

DELETE FROM kv WHERE "key" = 'instance_id';

COMMIT;
PRAGMA foreign_keys=ON;
