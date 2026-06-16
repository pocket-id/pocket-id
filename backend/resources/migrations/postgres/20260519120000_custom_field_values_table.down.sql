CREATE TEMP TABLE custom_field_migration_map AS
SELECT
    field.value->>'id' AS custom_field_id,
    field.value->>'key' AS key
FROM app_config_variables
CROSS JOIN LATERAL jsonb_array_elements(app_config_variables.value::jsonb) AS field(value)
WHERE app_config_variables.key = 'customFields';

ALTER TABLE custom_field_values RENAME CONSTRAINT custom_field_values_unique TO custom_field_values_custom_field_id_unique;

ALTER TABLE custom_field_values ADD COLUMN key VARCHAR(255);

UPDATE custom_field_values
SET key = custom_field_migration_map.key
FROM custom_field_migration_map
WHERE custom_field_migration_map.custom_field_id = custom_field_values.custom_field_id;

UPDATE custom_field_values
SET key = custom_field_id
WHERE key IS NULL;

ALTER TABLE custom_field_values ALTER COLUMN key SET NOT NULL;
ALTER TABLE custom_field_values DROP CONSTRAINT custom_field_values_custom_field_id_unique;
ALTER TABLE custom_field_values DROP COLUMN custom_field_id;
ALTER TABLE custom_field_values ADD CONSTRAINT custom_claims_unique UNIQUE (key, user_id, user_group_id);

ALTER TABLE custom_field_values RENAME TO custom_claims;

DROP TABLE custom_field_migration_map;

DELETE FROM app_config_variables WHERE key = 'customFields';
