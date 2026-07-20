-- Recreate the standalone config table with the same schema it had before it was frozen
CREATE TABLE app_config_variables
(
    key   VARCHAR(100) NOT NULL PRIMARY KEY,
    value TEXT NOT NULL
);

-- Populate it from the frozen JSON document stored in the "kv" table
-- json_each expands the JSON object back into one row per key/value pair.
INSERT INTO app_config_variables (key, value)
SELECT je.key, je.value
FROM kv, json_each_text(kv."value"::json) AS je(key, value)
WHERE kv."key" = 'config_migrated';

-- Remove the frozen config from the "kv" table
DELETE FROM kv WHERE "key" = 'config_migrated';
