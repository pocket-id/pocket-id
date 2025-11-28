alter table custom_claims add column is_ldap boolean not null default false;
alter table custom_claims drop constraint if exists custom_claims_unique;
alter table custom_claims add constraint custom_claims_unique unique (key, user_id, user_group_id, is_ldap);
