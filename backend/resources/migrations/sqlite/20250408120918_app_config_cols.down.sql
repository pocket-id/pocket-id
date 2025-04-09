ALTER TABLE app_config_variables ADD type VARCHAR(20) NOT NULL,;
ALTER TABLE app_config_variables ADD is_public BOOLEAN DEFAULT FALSE NOT NULL,;
ALTER TABLE app_config_variables ADD is_internal BOOLEAN DEFAULT FALSE NOT NULL,;
ALTER TABLE app_config_variables ADD default_value TEXT;
