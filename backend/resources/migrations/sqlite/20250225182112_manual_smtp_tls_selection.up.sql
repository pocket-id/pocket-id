UPDATE app_config_variables
SET value = CASE
                WHEN value = 'true' AND (SELECT value FROM app_config_variables WHERE key = 'smtpPort' LIMIT 1) = '587' THEN 'starttls'
                WHEN value = 'true' THEN 'tls'
                ELSE 'none'
    END
WHERE key = 'smtpTls';