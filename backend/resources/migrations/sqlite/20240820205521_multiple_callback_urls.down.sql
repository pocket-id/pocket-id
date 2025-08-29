PRAGMA foreign_keys=OFF;
BEGIN;
create table oidc_clients
(
    id            TEXT not null primary key,
    created_at    DATETIME,
    name          TEXT,
    secret        TEXT,
    callback_url  TEXT,
    image_type    TEXT,
    created_by_id TEXT
        references users
);

insert into oidc_clients(id, created_at, name, secret, callback_url, image_type, created_by_id)
select id,
       created_at,
       name,
       secret,
       json_extract(callback_urls, '$[0]'),
       image_type,
       created_by_id
from oidc_clients_dg_tmp;

drop table oidc_clients_dg_tmp;
COMMIT;
PRAGMA foreign_keys=ON;
