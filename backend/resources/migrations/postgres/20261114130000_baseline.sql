-- +goose Up

CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE api_keys (
    id uuid NOT NULL,
    name character varying(255) NOT NULL,
    key character varying(255) NOT NULL,
    description text,
    expires_at timestamp with time zone NOT NULL,
    last_used_at timestamp with time zone,
    created_at timestamp with time zone,
    user_id uuid,
    expiration_email_sent boolean DEFAULT false NOT NULL,
    scopes jsonb DEFAULT '[]'::jsonb NOT NULL,
    previous_key character varying(255),
    previous_key_expires_at timestamp with time zone
);

CREATE TABLE api_permissions (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    api_id uuid NOT NULL,
    key text NOT NULL,
    name text NOT NULL,
    description text,
    allowed_for_cimd_clients boolean DEFAULT false NOT NULL
);

CREATE TABLE apis (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone,
    name text NOT NULL,
    audience text NOT NULL,
    allow_cimd_clients boolean DEFAULT false NOT NULL
);

CREATE TABLE audit_logs (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    event character varying(100) NOT NULL,
    ip_address inet,
    data jsonb NOT NULL,
    user_id uuid,
    user_agent text NOT NULL,
    country character varying(100),
    city character varying(100),
    actor_type text,
    actor_id text,
    actor_name text
);

CREATE TABLE custom_claims (
    id uuid NOT NULL,
    created_at timestamp with time zone,
    key character varying(255) NOT NULL,
    value text NOT NULL,
    user_id uuid,
    user_group_id uuid,
    CONSTRAINT custom_claims_check CHECK (((user_id IS NOT NULL) OR (user_group_id IS NOT NULL)))
);

CREATE TABLE interaction_sessions (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    consent_required boolean DEFAULT false NOT NULL,
    reauthentication_required boolean DEFAULT false NOT NULL,
    authentication_required boolean DEFAULT false NOT NULL,
    account_selection_required boolean DEFAULT false NOT NULL,
    scopes jsonb DEFAULT '[]'::jsonb NOT NULL,
    client_id text NOT NULL,
    user_id uuid,
    requested_at timestamp with time zone NOT NULL,
    reauthenticated_at timestamp with time zone,
    parameters jsonb DEFAULT '{}'::jsonb NOT NULL
);

CREATE TABLE kv (
    key text NOT NULL,
    value text NOT NULL
);

CREATE TABLE oauth2_jtis (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    jti text NOT NULL,
    expires_at timestamp with time zone NOT NULL
);

CREATE TABLE oauth2_sessions (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    kind text NOT NULL,
    key text NOT NULL,
    request_id text NOT NULL,
    access_token_signature text DEFAULT ''::text NOT NULL,
    active boolean DEFAULT true NOT NULL,
    request_data jsonb NOT NULL,
    expires_at timestamp with time zone,
    client_id text NOT NULL,
    rotated_at timestamp with time zone,
    CONSTRAINT chk_oauth2_sessions_client_id CHECK ((client_id = (request_data ->> 'client_id'::text)))
);

CREATE TABLE oidc_claim_mapping_policies (
    id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    name text NOT NULL,
    is_default boolean DEFAULT false NOT NULL,
    claim_mappings jsonb NOT NULL
);

CREATE TABLE oidc_clients (
    id text NOT NULL,
    created_at timestamp with time zone NOT NULL,
    name character varying(255),
    callback_urls jsonb,
    image_type character varying(10),
    created_by_id uuid,
    is_public boolean DEFAULT false,
    pkce_enabled boolean DEFAULT false,
    logout_callback_urls jsonb,
    credentials jsonb,
    launch_url text,
    requires_reauthentication boolean DEFAULT false NOT NULL,
    dark_image_type text,
    is_group_restricted boolean DEFAULT false NOT NULL,
    requires_pushed_authorization_requests boolean DEFAULT false NOT NULL,
    skip_consent boolean DEFAULT false NOT NULL,
    pkce_supported boolean DEFAULT false NOT NULL,
    notes text DEFAULT ''::text NOT NULL,
    client_type text DEFAULT 'standard'::text NOT NULL,
    metadata_expires_at timestamp with time zone,
    metadata_grant_types jsonb,
    access_token_duration_minutes bigint DEFAULT 60 NOT NULL,
    refresh_token_duration_minutes bigint DEFAULT 43200 NOT NULL,
    backchannel_logout_url text DEFAULT ''::text NOT NULL,
    claim_mapping_policy_id uuid,
    disabled boolean DEFAULT false NOT NULL
);

CREATE TABLE oidc_clients_allowed_api_permissions (
    oidc_client_id text NOT NULL,
    api_permission_id uuid NOT NULL,
    subject_type text NOT NULL,
    CONSTRAINT oidc_clients_allowed_api_permissions_subject_type_check CHECK ((subject_type = ANY (ARRAY['user'::text, 'client'::text])))
);

CREATE TABLE oidc_clients_allowed_apis (
    oidc_client_id text NOT NULL,
    api_id uuid NOT NULL,
    subject_type text NOT NULL,
    CONSTRAINT oidc_clients_allowed_apis_subject_type_check CHECK ((subject_type = ANY (ARRAY['user'::text, 'client'::text])))
);

CREATE TABLE oidc_clients_allowed_user_groups (
    user_group_id uuid NOT NULL,
    oidc_client_id text NOT NULL
);

CREATE TABLE oidc_custom_scopes (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    key text NOT NULL,
    name text NOT NULL,
    description text NOT NULL
);

CREATE TABLE reauthentication_tokens (
    id text NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    token text NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    user_id uuid NOT NULL
);

CREATE TABLE scim_service_providers (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    endpoint text NOT NULL,
    token text NOT NULL,
    last_synced_at timestamp with time zone,
    oidc_client_id text NOT NULL
);

CREATE TABLE storage (
    path text NOT NULL,
    data bytea NOT NULL,
    size bigint NOT NULL,
    mod_time timestamp with time zone NOT NULL,
    created_at timestamp with time zone NOT NULL
);

CREATE TABLE user_authorized_oidc_clients (
    scope jsonb DEFAULT '[]'::jsonb NOT NULL,
    user_id uuid NOT NULL,
    client_id text NOT NULL,
    last_used_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE TABLE user_groups (
    id uuid NOT NULL,
    created_at timestamp with time zone,
    friendly_name character varying(255) NOT NULL,
    name character varying(255) NOT NULL,
    ldap_id text,
    updated_at timestamp with time zone,
    notes text DEFAULT ''::text NOT NULL
);

CREATE TABLE user_groups_users (
    user_id uuid NOT NULL,
    user_group_id uuid NOT NULL
);

CREATE TABLE users (
    id uuid NOT NULL,
    created_at timestamp with time zone,
    username citext NOT NULL COLLATE pg_catalog."C",
    email character varying(255),
    first_name character varying(100) NOT NULL,
    last_name character varying(100) NOT NULL,
    is_admin boolean DEFAULT false NOT NULL,
    ldap_id text,
    locale text,
    disabled boolean DEFAULT false NOT NULL,
    display_name text NOT NULL,
    updated_at timestamp with time zone,
    email_verified boolean DEFAULT false NOT NULL,
    email_verification_deadline timestamp with time zone
);

CREATE TABLE webauthn_credentials (
    id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    name character varying(255) NOT NULL,
    credential_id bytea NOT NULL,
    public_key bytea NOT NULL,
    attestation_type character varying(20) NOT NULL,
    transport jsonb NOT NULL,
    user_id uuid,
    backup_eligible boolean DEFAULT false NOT NULL,
    backup_state boolean DEFAULT false NOT NULL,
    aaguid text DEFAULT '00000000-0000-0000-0000-000000000000'::text NOT NULL,
    attestation_verified boolean DEFAULT false NOT NULL
);

CREATE TABLE webauthn_sessions (
    id uuid NOT NULL,
    created_at timestamp with time zone,
    challenge character varying(255) NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    user_verification character varying(255) NOT NULL,
    credential_params jsonb DEFAULT '[]'::jsonb NOT NULL,
    extensions jsonb DEFAULT '{}'::jsonb NOT NULL
);

ALTER TABLE ONLY api_keys
    ADD CONSTRAINT api_keys_key_key UNIQUE (key);

ALTER TABLE ONLY api_keys
    ADD CONSTRAINT api_keys_pkey PRIMARY KEY (id);

ALTER TABLE ONLY api_permissions
    ADD CONSTRAINT api_permissions_api_id_key_key UNIQUE (api_id, key);

ALTER TABLE ONLY api_permissions
    ADD CONSTRAINT api_permissions_pkey PRIMARY KEY (id);

ALTER TABLE ONLY apis
    ADD CONSTRAINT apis_audience_key UNIQUE (audience);

ALTER TABLE ONLY apis
    ADD CONSTRAINT apis_pkey PRIMARY KEY (id);

ALTER TABLE ONLY audit_logs
    ADD CONSTRAINT audit_logs_pkey PRIMARY KEY (id);

ALTER TABLE ONLY custom_claims
    ADD CONSTRAINT custom_claims_pkey PRIMARY KEY (id);

ALTER TABLE ONLY custom_claims
    ADD CONSTRAINT custom_claims_unique UNIQUE (key, user_id, user_group_id);

ALTER TABLE ONLY interaction_sessions
    ADD CONSTRAINT interaction_sessions_pkey PRIMARY KEY (id);

ALTER TABLE ONLY kv
    ADD CONSTRAINT kv_pkey PRIMARY KEY (key);

ALTER TABLE ONLY oauth2_jtis
    ADD CONSTRAINT oauth2_jtis_jti_key UNIQUE (jti);

ALTER TABLE ONLY oauth2_jtis
    ADD CONSTRAINT oauth2_jtis_pkey PRIMARY KEY (id);

ALTER TABLE ONLY oauth2_sessions
    ADD CONSTRAINT oauth2_sessions_pkey PRIMARY KEY (id);

ALTER TABLE ONLY oidc_claim_mapping_policies
    ADD CONSTRAINT oidc_claim_mapping_policies_pkey PRIMARY KEY (id);

ALTER TABLE ONLY oidc_clients_allowed_api_permissions
    ADD CONSTRAINT oidc_clients_allowed_api_permissions_pkey PRIMARY KEY (oidc_client_id, api_permission_id, subject_type);

ALTER TABLE ONLY oidc_clients_allowed_apis
    ADD CONSTRAINT oidc_clients_allowed_apis_pkey PRIMARY KEY (oidc_client_id, api_id, subject_type);

ALTER TABLE ONLY oidc_clients_allowed_user_groups
    ADD CONSTRAINT oidc_clients_allowed_user_groups_pkey PRIMARY KEY (oidc_client_id, user_group_id);

ALTER TABLE ONLY oidc_clients
    ADD CONSTRAINT oidc_clients_pkey PRIMARY KEY (id);

ALTER TABLE ONLY oidc_custom_scopes
    ADD CONSTRAINT oidc_custom_scopes_key_key UNIQUE (key);

ALTER TABLE ONLY oidc_custom_scopes
    ADD CONSTRAINT oidc_custom_scopes_pkey PRIMARY KEY (id);

ALTER TABLE ONLY reauthentication_tokens
    ADD CONSTRAINT reauthentication_tokens_pkey PRIMARY KEY (id);

ALTER TABLE ONLY reauthentication_tokens
    ADD CONSTRAINT reauthentication_tokens_token_key UNIQUE (token);

ALTER TABLE ONLY scim_service_providers
    ADD CONSTRAINT scim_service_providers_pkey PRIMARY KEY (id);

ALTER TABLE ONLY storage
    ADD CONSTRAINT storage_pkey PRIMARY KEY (path);

ALTER TABLE ONLY user_authorized_oidc_clients
    ADD CONSTRAINT user_authorized_oidc_clients_pkey PRIMARY KEY (user_id, client_id);

ALTER TABLE ONLY user_groups
    ADD CONSTRAINT user_groups_name_key UNIQUE (name);

ALTER TABLE ONLY user_groups
    ADD CONSTRAINT user_groups_pkey PRIMARY KEY (id);

ALTER TABLE ONLY user_groups_users
    ADD CONSTRAINT user_groups_users_pkey PRIMARY KEY (user_id, user_group_id);

ALTER TABLE ONLY users
    ADD CONSTRAINT users_email_key UNIQUE (email);

ALTER TABLE ONLY users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);

ALTER TABLE ONLY users
    ADD CONSTRAINT users_username_key UNIQUE (username);

ALTER TABLE ONLY webauthn_credentials
    ADD CONSTRAINT webauthn_credentials_credential_id_key UNIQUE (credential_id);

ALTER TABLE ONLY webauthn_credentials
    ADD CONSTRAINT webauthn_credentials_pkey PRIMARY KEY (id);

ALTER TABLE ONLY webauthn_sessions
    ADD CONSTRAINT webauthn_sessions_challenge_key UNIQUE (challenge);

ALTER TABLE ONLY webauthn_sessions
    ADD CONSTRAINT webauthn_sessions_pkey PRIMARY KEY (id);

CREATE INDEX idx_api_keys_expires_at ON api_keys USING btree (expires_at);

CREATE UNIQUE INDEX idx_api_keys_previous_key ON api_keys USING btree (previous_key);

CREATE INDEX idx_api_permissions_api_id ON api_permissions USING btree (api_id);

CREATE INDEX idx_audit_logs_actor_id ON audit_logs USING btree (actor_id);

CREATE INDEX idx_audit_logs_client_name ON audit_logs USING btree (((data ->> 'clientName'::text)));

CREATE INDEX idx_audit_logs_country ON audit_logs USING btree (country);

CREATE INDEX idx_audit_logs_created_at ON audit_logs USING btree (created_at);

CREATE INDEX idx_audit_logs_event ON audit_logs USING btree (event);

CREATE INDEX idx_audit_logs_user_agent ON audit_logs USING btree (user_agent);

CREATE INDEX idx_audit_logs_user_id ON audit_logs USING btree (user_id);

CREATE INDEX idx_interaction_sessions_client_id ON interaction_sessions USING btree (client_id);

CREATE INDEX idx_interaction_sessions_user_id ON interaction_sessions USING btree (user_id);

CREATE INDEX idx_oauth2_jtis_expires_at ON oauth2_jtis USING btree (expires_at);

CREATE INDEX idx_oauth2_sessions_client_subject ON oauth2_sessions USING btree (client_id, ((request_data #>> '{session,subject}'::text[])), kind, active);

CREATE INDEX idx_oauth2_sessions_expires_at ON oauth2_sessions USING btree (expires_at);

CREATE UNIQUE INDEX idx_oauth2_sessions_kind_key ON oauth2_sessions USING btree (kind, key);

CREATE INDEX idx_oauth2_sessions_kind_request ON oauth2_sessions USING btree (kind, request_id, active, rotated_at);

CREATE UNIQUE INDEX idx_oidc_claim_mapping_policies_default ON oidc_claim_mapping_policies USING btree (is_default) WHERE is_default;

CREATE INDEX idx_oidc_clients_allowed_apis_api_id ON oidc_clients_allowed_apis USING btree (api_id);

CREATE INDEX idx_reauthentication_tokens_expires_at ON reauthentication_tokens USING btree (expires_at);

CREATE INDEX idx_webauthn_sessions_expires_at ON webauthn_sessions USING btree (expires_at);

CREATE UNIQUE INDEX user_groups_ldap_id ON user_groups USING btree (ldap_id);

CREATE UNIQUE INDEX users_ldap_id ON users USING btree (ldap_id);

ALTER TABLE ONLY api_keys
    ADD CONSTRAINT api_keys_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE ONLY api_permissions
    ADD CONSTRAINT api_permissions_api_id_fkey FOREIGN KEY (api_id) REFERENCES apis(id) ON DELETE CASCADE;

ALTER TABLE ONLY audit_logs
    ADD CONSTRAINT audit_logs_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE ONLY custom_claims
    ADD CONSTRAINT custom_claims_user_group_id_fkey FOREIGN KEY (user_group_id) REFERENCES user_groups(id) ON DELETE CASCADE;

ALTER TABLE ONLY custom_claims
    ADD CONSTRAINT custom_claims_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE ONLY oauth2_sessions
    ADD CONSTRAINT fk_oauth2_sessions_client_id FOREIGN KEY (client_id) REFERENCES oidc_clients(id) ON DELETE CASCADE;

ALTER TABLE ONLY interaction_sessions
    ADD CONSTRAINT interaction_sessions_client_id_fkey FOREIGN KEY (client_id) REFERENCES oidc_clients(id) ON DELETE CASCADE;

ALTER TABLE ONLY interaction_sessions
    ADD CONSTRAINT interaction_sessions_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE ONLY oidc_clients_allowed_api_permissions
    ADD CONSTRAINT oidc_clients_allowed_api_permissions_api_permission_id_fkey FOREIGN KEY (api_permission_id) REFERENCES api_permissions(id) ON DELETE CASCADE;

ALTER TABLE ONLY oidc_clients_allowed_api_permissions
    ADD CONSTRAINT oidc_clients_allowed_api_permissions_oidc_client_id_fkey FOREIGN KEY (oidc_client_id) REFERENCES oidc_clients(id) ON DELETE CASCADE;

ALTER TABLE ONLY oidc_clients_allowed_apis
    ADD CONSTRAINT oidc_clients_allowed_apis_api_id_fkey FOREIGN KEY (api_id) REFERENCES apis(id) ON DELETE CASCADE;

ALTER TABLE ONLY oidc_clients_allowed_apis
    ADD CONSTRAINT oidc_clients_allowed_apis_oidc_client_id_fkey FOREIGN KEY (oidc_client_id) REFERENCES oidc_clients(id) ON DELETE CASCADE;

ALTER TABLE ONLY oidc_clients_allowed_user_groups
    ADD CONSTRAINT oidc_clients_allowed_user_groups_oidc_client_id_fkey FOREIGN KEY (oidc_client_id) REFERENCES oidc_clients(id) ON DELETE CASCADE;

ALTER TABLE ONLY oidc_clients_allowed_user_groups
    ADD CONSTRAINT oidc_clients_allowed_user_groups_user_group_id_fkey FOREIGN KEY (user_group_id) REFERENCES user_groups(id) ON DELETE CASCADE;

ALTER TABLE ONLY oidc_clients
    ADD CONSTRAINT oidc_clients_claim_mapping_policy_id_fkey FOREIGN KEY (claim_mapping_policy_id) REFERENCES oidc_claim_mapping_policies(id) ON DELETE SET NULL;

ALTER TABLE ONLY oidc_clients
    ADD CONSTRAINT oidc_clients_created_by_id_fkey FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE ONLY reauthentication_tokens
    ADD CONSTRAINT reauthentication_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE ONLY scim_service_providers
    ADD CONSTRAINT scim_service_providers_oidc_client_id_fkey FOREIGN KEY (oidc_client_id) REFERENCES oidc_clients(id) ON DELETE CASCADE;

ALTER TABLE ONLY user_authorized_oidc_clients
    ADD CONSTRAINT user_authorized_oidc_clients_client_id_fkey FOREIGN KEY (client_id) REFERENCES oidc_clients(id) ON DELETE CASCADE;

ALTER TABLE ONLY user_authorized_oidc_clients
    ADD CONSTRAINT user_authorized_oidc_clients_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE ONLY user_groups_users
    ADD CONSTRAINT user_groups_users_user_group_id_fkey FOREIGN KEY (user_group_id) REFERENCES user_groups(id) ON DELETE CASCADE;

ALTER TABLE ONLY user_groups_users
    ADD CONSTRAINT user_groups_users_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE ONLY webauthn_credentials
    ADD CONSTRAINT webauthn_credentials_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
