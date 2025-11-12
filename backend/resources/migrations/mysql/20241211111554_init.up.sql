CREATE TABLE `app_config_variables` (
    `key` VARCHAR(100) NOT NULL,
    `value` TEXT NOT NULL,
    PRIMARY KEY (`key`)
);

CREATE TABLE `audit_logs` (
    `id` CHAR(36) NOT NULL,
    `created_at` TIMESTAMP NULL,
    `event` VARCHAR(100) NOT NULL,
    `ip_address` VARCHAR(45) NULL,
    `data` JSON NOT NULL,
    `user_id` CHAR(36) NULL,
    `user_agent` TEXT NULL,
    `country` VARCHAR(100) NULL,
    `city` VARCHAR(100) NULL,
    PRIMARY KEY (`id`)
);

CREATE TABLE `custom_claims` (
    `id` CHAR(36) NOT NULL,
    `created_at` TIMESTAMP NULL,
    `key` VARCHAR(255) NOT NULL,
    `value` TEXT NOT NULL,
    `user_id` CHAR(36) NULL,
    `user_group_id` CHAR(36) NULL,
    PRIMARY KEY (`id`)
);

CREATE TABLE `user_groups` (
    `id` CHAR(36) NOT NULL,
    `created_at` TIMESTAMP NULL,
    `friendly_name` VARCHAR(255) NOT NULL,
    `name` VARCHAR(255) NOT NULL,
    `ldap_id` TEXT NULL,
    PRIMARY KEY (`id`)
);

CREATE TABLE `users` (
    `id` CHAR(36) NOT NULL,
    `created_at` TIMESTAMP NULL,
    `username` VARCHAR(255) NOT NULL,
    `email` VARCHAR(255) NULL,
    `first_name` VARCHAR(100) NULL,
    `last_name` VARCHAR(100) NULL,
    `is_admin` BOOLEAN NOT NULL DEFAULT FALSE,
    `ldap_id` TEXT NULL,
    `locale` TEXT NULL,
    `disabled` BOOLEAN NOT NULL DEFAULT FALSE,
    `display_name` TEXT NOT NULL,
    PRIMARY KEY (`id`)
);

CREATE TABLE `oidc_clients` (
    `id` VARCHAR(255) NOT NULL,
    `created_at` TIMESTAMP NULL,
    `name` VARCHAR(255) NULL,
    `secret` TEXT NULL,
    `callback_urls` JSON NULL,
    `image_type` VARCHAR(10) NULL,
    `created_by_id` CHAR(36) NULL,
    `is_public` BOOLEAN DEFAULT FALSE,
    `pkce_enabled` BOOLEAN DEFAULT FALSE,
    `logout_callback_urls` JSON NULL,
    `credentials` JSON NULL,
    `launch_url` TEXT NULL,
    `requires_reauthentication` BOOLEAN NOT NULL DEFAULT FALSE,
    `dark_image_type` TEXT NULL,
    PRIMARY KEY (`id`)
);

CREATE TABLE `one_time_access_tokens` (
    `id` CHAR(36) NOT NULL,
    `created_at` TIMESTAMP NULL,
    `token` VARCHAR(255) NOT NULL,
    `expires_at` TIMESTAMP NOT NULL,
    `user_id` CHAR(36) NOT NULL,
    PRIMARY KEY (`id`)
);

CREATE TABLE `oidc_authorization_codes` (
    `id` CHAR(36) NOT NULL,
    `created_at` TIMESTAMP NULL,
    `code` VARCHAR(255) NOT NULL,
    `scope` TEXT NOT NULL,
    `nonce` VARCHAR(255) NULL,
    `expires_at` TIMESTAMP NOT NULL,
    `user_id` CHAR(36) NOT NULL,
    `client_id` VARCHAR(255) NOT NULL,
    `code_challenge` VARCHAR(255) NULL,
    `code_challenge_method_sha256` BOOLEAN NULL,
    PRIMARY KEY (`id`)
);

CREATE TABLE `user_groups_users` (
    `user_id` CHAR(36) NOT NULL,
    `user_group_id` CHAR(36) NOT NULL,
    PRIMARY KEY (`user_id`, `user_group_id`)
);

CREATE TABLE `user_authorized_oidc_clients` (
    `scope` VARCHAR(255) NULL,
    `user_id` CHAR(36) NOT NULL,
    `client_id` VARCHAR(255) NOT NULL,
    `last_used_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`user_id`, `client_id`)
);

CREATE TABLE `webauthn_credentials` (
    `id` CHAR(36) NOT NULL,
    `created_at` TIMESTAMP NULL,
    `name` VARCHAR(255) NOT NULL,
    `credential_id` BLOB NOT NULL,
    `public_key` BLOB NOT NULL,
    `attestation_type` VARCHAR(20) NOT NULL,
    `transport` JSON NOT NULL,
    `user_id` CHAR(36) NULL,
    `backup_eligible` BOOLEAN NOT NULL DEFAULT FALSE,
    `backup_state` BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (`id`)
);

CREATE TABLE `webauthn_sessions` (
    `id` CHAR(36) NOT NULL,
    `created_at` TIMESTAMP NULL,
    `challenge` VARCHAR(255) NOT NULL,
    `expires_at` TIMESTAMP NOT NULL,
    `user_verification` VARCHAR(255) NOT NULL,
    `credential_params` JSON NOT NULL DEFAULT ('[]'),
    PRIMARY KEY (`id`)
);

CREATE TABLE `api_keys` (
    `id` CHAR(36) NOT NULL,
    `name` VARCHAR(255) NOT NULL,
    `key` VARCHAR(255) NOT NULL,
    `description` TEXT NULL,
    `expires_at` TIMESTAMP NOT NULL,
    `last_used_at` TIMESTAMP NULL,
    `created_at` TIMESTAMP NULL,
    `user_id` CHAR(36) NULL,
    `expiration_email_sent` BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (`id`)
);

CREATE TABLE `signup_tokens` (
    `id` CHAR(36) NOT NULL,
    `created_at` TIMESTAMP NOT NULL,
    `token` VARCHAR(255) NOT NULL,
    `expires_at` TIMESTAMP NOT NULL,
    `usage_limit` INT NOT NULL DEFAULT 1,
    `usage_count` INT NOT NULL DEFAULT 0,
    PRIMARY KEY (`id`)
);

CREATE TABLE `kv` (
    `key` TEXT NOT NULL,
    `value` TEXT NULL,
    PRIMARY KEY (`key`(255))
);

CREATE TABLE `reauthentication_tokens` (
    `id` TEXT NOT NULL,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `token` TEXT NOT NULL,
    `expires_at` TIMESTAMP NOT NULL,
    `user_id` CHAR(36) NOT NULL,
    PRIMARY KEY (`id`(255))
);

CREATE TABLE `oidc_refresh_tokens` (
    `id` CHAR(36) NOT NULL,
    `created_at` TIMESTAMP NULL,
    `token` VARCHAR(255) NOT NULL,
    `expires_at` TIMESTAMP NOT NULL,
    `scope` TEXT NOT NULL,
    `user_id` CHAR(36) NOT NULL,
    `client_id` VARCHAR(255) NOT NULL,
    PRIMARY KEY (`id`)
);

CREATE TABLE `oidc_device_codes` (
    `id` CHAR(36) NOT NULL,
    `created_at` TIMESTAMP NULL,
    `device_code` TEXT NOT NULL,
    `user_code` TEXT NOT NULL,
    `scope` TEXT NOT NULL,
    `expires_at` TIMESTAMP NOT NULL,
    `is_authorized` BOOLEAN NOT NULL DEFAULT FALSE,
    `user_id` CHAR(36) NULL,
    `client_id` VARCHAR(255) NOT NULL,
    PRIMARY KEY (`id`)
);

CREATE TABLE `oidc_clients_allowed_user_groups` (
    `user_group_id` CHAR(36) NOT NULL,
    `oidc_client_id` VARCHAR(255) NOT NULL,
    PRIMARY KEY (`oidc_client_id`, `user_group_id`)
);

ALTER TABLE `audit_logs` ADD FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE;

CREATE INDEX `idx_audit_logs_event` ON `audit_logs` (`event`);
CREATE INDEX `idx_audit_logs_user_id` ON `audit_logs` (`user_id`);
CREATE INDEX `idx_audit_logs_client_name` ON `audit_logs` ((CAST(JSON_UNQUOTE(JSON_EXTRACT(`data`, '$.clientName')) AS CHAR(255))));
CREATE INDEX `idx_audit_logs_country` ON `audit_logs` (`country`);
CREATE INDEX `idx_audit_logs_created_at` ON `audit_logs` (`created_at`);
CREATE INDEX `idx_audit_logs_user_agent` ON `audit_logs` (`user_agent`(255));
ALTER TABLE `custom_claims` ADD FOREIGN KEY (`user_group_id`) REFERENCES `user_groups`(`id`) ON DELETE CASCADE;
ALTER TABLE `custom_claims` ADD FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE;
CREATE UNIQUE INDEX `custom_claims_unique` ON `custom_claims` (`key`, `user_id`, `user_group_id`);
CREATE UNIQUE INDEX `user_groups_name_key` ON `user_groups` (`name`);
CREATE UNIQUE INDEX `user_groups_ldap_id` ON `user_groups` (`ldap_id`(255));
CREATE UNIQUE INDEX `users_email_key` ON `users` (`email`);
CREATE UNIQUE INDEX `users_ldap_id` ON `users` (`ldap_id`(255));
CREATE UNIQUE INDEX `users_username_key` ON `users` (`username`);
ALTER TABLE `oidc_clients` ADD FOREIGN KEY (`created_by_id`) REFERENCES `users`(`id`) ON DELETE SET NULL;
ALTER TABLE `one_time_access_tokens` ADD FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE;
CREATE UNIQUE INDEX `one_time_access_tokens_token_key` ON `one_time_access_tokens` (`token`);
ALTER TABLE `oidc_authorization_codes` ADD FOREIGN KEY (`client_id`) REFERENCES `oidc_clients`(`id`) ON DELETE CASCADE;
ALTER TABLE `oidc_authorization_codes` ADD FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE;
CREATE UNIQUE INDEX `oidc_authorization_codes_code_key` ON `oidc_authorization_codes` (`code`);
ALTER TABLE `user_groups_users` ADD FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE;
ALTER TABLE `user_groups_users` ADD FOREIGN KEY (`user_group_id`) REFERENCES `user_groups`(`id`) ON DELETE CASCADE;
ALTER TABLE `user_authorized_oidc_clients` ADD FOREIGN KEY (`client_id`) REFERENCES `oidc_clients`(`id`) ON DELETE CASCADE;
ALTER TABLE `user_authorized_oidc_clients` ADD FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE;
ALTER TABLE `webauthn_credentials` ADD FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE;
CREATE UNIQUE INDEX `webauthn_credentials_credential_id_key` ON `webauthn_credentials` (`credential_id`(255));
ALTER TABLE `webauthn_sessions` ADD UNIQUE INDEX `webauthn_sessions_challenge_key` (`challenge`);
ALTER TABLE `api_keys` ADD FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE;
CREATE UNIQUE INDEX `api_keys_key_key` ON `api_keys` (`key`);
CREATE UNIQUE INDEX `signup_tokens_token_key` ON `signup_tokens` (`token`);
CREATE INDEX `idx_signup_tokens_expires_at` ON `signup_tokens` (`expires_at`);
ALTER TABLE `reauthentication_tokens` ADD FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE;
CREATE UNIQUE INDEX `reauthentication_tokens_token_key` ON `reauthentication_tokens` (`token`(255));
ALTER TABLE `oidc_refresh_tokens` ADD FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE;
ALTER TABLE `oidc_refresh_tokens` ADD FOREIGN KEY (`client_id`) REFERENCES `oidc_clients`(`id`) ON DELETE CASCADE;
CREATE UNIQUE INDEX `oidc_refresh_tokens_token_key` ON `oidc_refresh_tokens` (`token`);
ALTER TABLE `oidc_device_codes` ADD FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE;
ALTER TABLE `oidc_device_codes` ADD FOREIGN KEY (`client_id`) REFERENCES `oidc_clients`(`id`) ON DELETE CASCADE;
CREATE UNIQUE INDEX `oidc_device_codes_device_code_key` ON `oidc_device_codes` (`device_code`(255));
CREATE UNIQUE INDEX `oidc_device_codes_user_code_key` ON `oidc_device_codes` (`user_code`(255));
ALTER TABLE `oidc_clients_allowed_user_groups` ADD FOREIGN KEY (`user_group_id`) REFERENCES `user_groups`(`id`) ON DELETE CASCADE;
ALTER TABLE `oidc_clients_allowed_user_groups` ADD FOREIGN KEY (`oidc_client_id`) REFERENCES `oidc_clients`(`id`) ON DELETE CASCADE;
