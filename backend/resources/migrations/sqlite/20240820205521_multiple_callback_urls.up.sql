PRAGMA foreign_keys=OFF;
BEGIN;
create table oidc_clients_dg_tmp
(
    id            TEXT not null primary key,
    created_at    DATETIME,
    name          TEXT,
    secret        TEXT,
    callback_urls BLOB,
    image_type    TEXT,
    created_by_id TEXT
        references users
);

insert into oidc_clients_dg_tmp(id, created_at, name, secret, callback_urls, image_type, created_by_id)
select id,
       created_at,
       name,
       secret,
       CAST('["' || callback_url || '"]' AS BLOB),
       image_type,
       created_by_id
from oidc_clients;

drop table oidc_clients;

alter table oidc_clients_dg_tmp
    rename to oidc_clients;
COMMIT;
PRAGMA foreign_keys=ON;
