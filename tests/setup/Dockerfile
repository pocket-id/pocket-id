# Modified version of lldap/lldap that seeds lldap with test data
FROM lldap/lldap:2025-05-19

WORKDIR /app

COPY seed-lldap.sh .
RUN chmod +x ./seed-lldap.sh
RUN cp lldap_set_password /bin

ENTRYPOINT /docker-entrypoint.sh run --config-file /data/lldap_config.toml & ./seed-lldap.sh && wait
