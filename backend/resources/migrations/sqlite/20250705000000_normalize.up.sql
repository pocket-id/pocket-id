-- Normalize (form NFC) all existing values in the database
UPDATE api_keys SET
    name = normalize(name, 'nfc'),
    description = normalize(description, 'nfc');

UPDATE app_config_variables SET
    "value" = normalize("value", 'nfc')
WHERE "key" = 'appName';

UPDATE custom_claims SET
    "key" = normalize("key", 'nfc'),
    "value" = normalize("value", 'nfc');

UPDATE oidc_clients SET
    name = normalize(name, 'nfc');

UPDATE users SET
    username = normalize(username, 'nfc'),
    email = normalize(email, 'nfc'),
    first_name = normalize(first_name, 'nfc'),
    last_name = normalize(last_name, 'nfc');

UPDATE user_groups SET
    friendly_name = normalize(friendly_name, 'nfc'),
    "name" = normalize("name", 'nfc');
