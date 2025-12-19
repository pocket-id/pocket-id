alter table custom_claims drop constraint if exists custom_claims_unique;
alter table custom_claims add constraint custom_claims_unique unique (key, user_id, user_group_id);
alter table custom_claims drop column is_ldap;
delete from app_config_variables where key = 'ldapExtraAttributes';
