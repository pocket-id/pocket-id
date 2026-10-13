CREATE TABLE oidc_claim_mapping_policies (
    id UUID NOT NULL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    name TEXT NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    claim_mappings JSONB NOT NULL
);

INSERT INTO oidc_claim_mapping_policies (id, name, is_default, claim_mappings) VALUES (
    'f383d02d-69e1-4a28-96f8-7c5410c8ac8d',
    'Standard Policy',
    TRUE,
    '[{"claimName":"sub","sourceType":"user_field","sourceValue":"id","scope":["openid"],"accessToken":true,"idToken":true,"userInfo":true},{"claimName":"email","sourceType":"user_field","sourceValue":"email","scope":["email"],"accessToken":false,"idToken":true,"userInfo":true},{"claimName":"email_verified","sourceType":"user_field","sourceValue":"email_verified","scope":["email"],"accessToken":false,"idToken":true,"userInfo":true},{"claimName":"given_name","sourceType":"user_field","sourceValue":"first_name","scope":["profile"],"accessToken":false,"idToken":true,"userInfo":true},{"claimName":"family_name","sourceType":"user_field","sourceValue":"last_name","scope":["profile"],"accessToken":false,"idToken":true,"userInfo":true},{"claimName":"name","sourceType":"user_field","sourceValue":"full_name","scope":["profile"],"accessToken":false,"idToken":true,"userInfo":true},{"claimName":"display_name","sourceType":"user_field","sourceValue":"display_name","scope":["profile"],"accessToken":false,"idToken":true,"userInfo":true},{"claimName":"preferred_username","sourceType":"user_field","sourceValue":"username","scope":["profile"],"accessToken":false,"idToken":true,"userInfo":true},{"claimName":"locale","sourceType":"user_field","sourceValue":"locale","scope":["profile"],"accessToken":false,"idToken":true,"userInfo":true},{"claimName":"picture","sourceType":"user_field","sourceValue":"picture","scope":["profile"],"accessToken":false,"idToken":true,"userInfo":true},{"claimName":"groups","sourceType":"user_field","sourceValue":"groups","scope":["groups"],"accessToken":false,"idToken":true,"userInfo":true},{"claimName":"*","sourceType":"custom_claim","sourceValue":"*","scope":["profile"],"accessToken":false,"idToken":true,"userInfo":true}]'
);

ALTER TABLE oidc_clients ADD COLUMN claim_mapping_policy_id UUID REFERENCES oidc_claim_mapping_policies(id);
