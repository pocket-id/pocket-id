UPDATE app_config_variables AS target
SET value = CASE
    WHEN target.value = 'true' AND (SELECT value FROM app_config_variables WHERE key = 'smtpPort' LIMIT 1) = '587' THEN 'starttls'
    WHEN target.value = 'true' THEN 'tls'
    ELSE 'none'
END
    WHERE target.key = 'smtpTls';