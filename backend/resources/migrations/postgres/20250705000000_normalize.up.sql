-- Normalize (form NFC) all existing values in the database
DO $$
BEGIN
    -- This function is available only if the server's encoding is UTF8
    IF current_setting('server_encoding') = 'UTF8' THEN
        UPDATE api_keys SET
            name = normalize(name, NFC),
            description = normalize(description, NFC);

        UPDATE app_config_variables SET
            "value" = normalize("value", NFC)
        WHERE "key" = 'appName';

        UPDATE custom_claims SET
            "key" = normalize("key", NFC),
            "value" = normalize("value", NFC);

        UPDATE oidc_clients SET
            name = normalize(name, NFC);

        UPDATE users SET
            username = normalize(username, NFC),
            email = normalize(email, NFC),
            first_name = normalize(first_name, NFC),
            last_name = normalize(last_name, NFC);

        UPDATE user_groups SET
            friendly_name = normalize(friendly_name, NFC),
            "name" = normalize("name", NFC);
    ELSE
		RAISE NOTICE 'Skipping normalization: server_encoding is %', current_setting('server_encoding');
	END IF;
END;
$$ LANGUAGE plpgsql;