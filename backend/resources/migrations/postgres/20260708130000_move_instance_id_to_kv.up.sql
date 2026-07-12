-- Move the instance ID out of the standalone app config table and into the "kv" table
INSERT INTO kv ("key", "value")
SELECT 'instance_id', "value"
    FROM app_config_variables
    WHERE "key" = 'instanceId'
    ON CONFLICT ("key") DO NOTHING;

DELETE FROM app_config_variables WHERE "key" = 'instanceId';
