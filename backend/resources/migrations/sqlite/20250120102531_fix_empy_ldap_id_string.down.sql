UPDATE users SET ldap_id = '' WHERE ldap_id IS NULL;
UPDATE user_groups SET ldap_id = '' WHERE ldap_id IS NULL;