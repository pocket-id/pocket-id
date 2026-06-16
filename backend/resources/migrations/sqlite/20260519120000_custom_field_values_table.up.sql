ALTER TABLE custom_claims RENAME TO custom_field_values_old;

CREATE TEMP TABLE custom_field_migration_map AS
SELECT
    key,
    lower(hex(randomblob(4))) || '-' ||
    lower(hex(randomblob(2))) || '-' ||
    '4' || substr(lower(hex(randomblob(2))), 2) || '-' ||
    substr('89ab', abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))), 2) || '-' ||
    lower(hex(randomblob(6))) AS custom_field_id,
    MAX(user_id IS NOT NULL) AS has_user_values,
    MAX(user_group_id IS NOT NULL) AS has_group_values
FROM custom_field_values_old
GROUP BY key;

INSERT INTO app_config_variables (key, value)
VALUES (
    'customFields',
    COALESCE(
        (
            SELECT json_group_array(
                json_object(
                    'id', custom_field_id,
                    'key', key,
                    'displayName', key,
                    'type', 'string',
                    'target',
                        CASE
                            WHEN has_user_values = 1 AND has_group_values = 1 THEN 'both'
                            WHEN has_user_values = 1 THEN 'user'
                            ELSE 'group'
                        END,
                    'required', json('false'),
                    'userEditable', json('false'),
                    'defaultValue', '',
                    'validationRegex', '',
                    'validationErrorMessage', ''
                )
            )
            FROM custom_field_migration_map
            ORDER BY key
        ),
        '[]'
    )
)
ON CONFLICT(key) DO UPDATE SET value = excluded.value
WHERE app_config_variables.value = '' OR app_config_variables.value = '[]';

CREATE TABLE custom_field_values
(
    id              TEXT NOT NULL PRIMARY KEY,
    created_at      DATETIME,
    custom_field_id TEXT NOT NULL,
    value           TEXT NOT NULL,

    user_id         TEXT,
    user_group_id   TEXT,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    FOREIGN KEY (user_group_id) REFERENCES user_groups (id) ON DELETE CASCADE,

    CONSTRAINT custom_field_values_unique UNIQUE (custom_field_id, user_id, user_group_id),
    CHECK (user_id IS NOT NULL OR user_group_id IS NOT NULL)
);

INSERT INTO custom_field_values (id, created_at, custom_field_id, value, user_id, user_group_id)
SELECT old.id, old.created_at, map.custom_field_id, old.value, old.user_id, old.user_group_id
FROM custom_field_values_old old
JOIN custom_field_migration_map map ON map.key = old.key;

DROP TABLE custom_field_values_old;
DROP TABLE custom_field_migration_map;

DELETE FROM app_config_variables WHERE key IN ('userCustomFields', 'userGroupCustomFields');
