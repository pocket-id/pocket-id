CREATE TEMP TABLE custom_field_migration_map AS
SELECT
    json_extract(json_each.value, '$.id') AS custom_field_id,
    json_extract(json_each.value, '$.key') AS key
FROM app_config_variables, json_each(app_config_variables.value)
WHERE app_config_variables.key = 'customFields';

ALTER TABLE custom_field_values RENAME TO custom_field_values_old;

CREATE TABLE custom_claims
(
    id            TEXT NOT NULL PRIMARY KEY,
    created_at    DATETIME,
    key           TEXT NOT NULL,
    value         TEXT NOT NULL,

    user_id       TEXT,
    user_group_id TEXT,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    FOREIGN KEY (user_group_id) REFERENCES user_groups (id) ON DELETE CASCADE,

    CONSTRAINT custom_claims_unique UNIQUE (key, user_id, user_group_id),
    CHECK (user_id IS NOT NULL OR user_group_id IS NOT NULL)
);

INSERT INTO custom_claims (id, created_at, key, value, user_id, user_group_id)
SELECT
    old.id,
    old.created_at,
    COALESCE(map.key, old.custom_field_id),
    old.value,
    old.user_id,
    old.user_group_id
FROM custom_field_values_old old
LEFT JOIN custom_field_migration_map map ON map.custom_field_id = old.custom_field_id;

DROP TABLE custom_field_values_old;
DROP TABLE custom_field_migration_map;

DELETE FROM app_config_variables WHERE key = 'customFields';
