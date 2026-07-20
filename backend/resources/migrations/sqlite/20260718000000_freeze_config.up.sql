PRAGMA foreign_keys=OFF;
BEGIN;

-- Freeze the app configuration
-- Encode every row of the standalone config table as a single JSON object (mapping key -> value) and store it in the "kv" table under the "config_migrated" key
--
-- json_group_object aggregates all rows into a JSON object
-- The "HAVING count(*) > 0" clause ensures that nothing is written to the "kv" table when the config table is empty
INSERT INTO kv ("key", "value")
SELECT 'config_migrated', json_group_object("key", "value")
FROM app_config_variables
HAVING count(*) > 0;

-- Drop the now-frozen standalone config table
DROP TABLE app_config_variables;

COMMIT;
PRAGMA foreign_keys=ON;
