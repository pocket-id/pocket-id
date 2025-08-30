ALTER TABLE public.audit_logs
    DROP CONSTRAINT IF EXISTS audit_logs_user_id_fkey,
    ADD CONSTRAINT audit_logs_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES public.users (id) ON DELETE CASCADE;

ALTER TABLE public.oidc_authorization_codes
    ADD CONSTRAINT oidc_authorization_codes_client_fk
        FOREIGN KEY (client_id) REFERENCES public.oidc_clients (id) ON DELETE CASCADE;