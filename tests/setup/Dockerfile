# Modified version of lldap/lldap that seeds lldap with test data
FROM lldap/lldap:2025-05-19

WORKDIR /app

RUN curl -o /bin/lldap-cli https://raw.githubusercontent.com/Zepmann/lldap-cli/e383494b4dd89ae4e028958b268e200fd85a7a64/lldap-cli

COPY seed-lldap.sh .
RUN chmod +x ./seed-lldap.sh /bin/lldap-cli
RUN cp lldap_set_password /bin

ENTRYPOINT /docker-entrypoint.sh run --config-file /data/lldap_config.toml & ./seed-lldap.sh && wait

