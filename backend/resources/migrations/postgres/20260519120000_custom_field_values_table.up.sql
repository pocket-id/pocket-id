ALTER TABLE custom_claims RENAME TO custom_field_values;
ALTER TABLE custom_field_values RENAME CONSTRAINT custom_claims_unique TO custom_field_values_key_unique;

ALTER TABLE custom_field_values ADD COLUMN custom_field_id VARCHAR(255);

CREATE TEMP TABLE custom_field_migration_map AS
SELECT
    key,
    SUBSTRING(md5(key) FROM 1 FOR 8) || '-' ||
    SUBSTRING(md5(key) FROM 9 FOR 4) || '-' ||
    '4' || SUBSTRING(md5(key) FROM 14 FOR 3) || '-' ||
    '8' || SUBSTRING(md5(key) FROM 18 FOR 3) || '-' ||
    SUBSTRING(md5(key) FROM 21 FOR 12) AS custom_field_id,
    BOOL_OR(user_id IS NOT NULL) AS has_user_values,
    BOOL_OR(user_group_id IS NOT NULL) AS has_group_values
FROM custom_field_values
GROUP BY key;

INSERT INTO app_config_variables (key, value)
SELECT
    'customFields',
    COALESCE(
        jsonb_agg(
            jsonb_build_object(
                'id', custom_field_id,
                'key', key,
                'displayName', key,
                'type', 'string',
                'target',
                    CASE
                        WHEN has_user_values AND has_group_values THEN 'both'
                        WHEN has_user_values THEN 'user'
                        ELSE 'group'
                    END,
                'required', false,
                'userEditable', false,
                'defaultValue', '',
                'validationRegex', '',
                'validationErrorMessage', ''
            )
            ORDER BY key
        )::TEXT,
        '[]'
    )
FROM custom_field_migration_map
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
WHERE app_config_variables.value = '' OR app_config_variables.value = '[]';

UPDATE custom_field_values
SET custom_field_id = custom_field_migration_map.custom_field_id
FROM custom_field_migration_map
WHERE custom_field_migration_map.key = custom_field_values.key;

ALTER TABLE custom_field_values ALTER COLUMN custom_field_id SET NOT NULL;
ALTER TABLE custom_field_values DROP CONSTRAINT custom_field_values_key_unique;
ALTER TABLE custom_field_values DROP COLUMN key;
ALTER TABLE custom_field_values ADD CONSTRAINT custom_field_values_unique UNIQUE (custom_field_id, user_id, user_group_id);

DROP TABLE custom_field_migration_map;

DELETE FROM app_config_variables WHERE key IN ('userCustomFields', 'userGroupCustomFields');
