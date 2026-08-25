FROM ghcr.io/sbekti/freeradius:v3.2.8

ARG DEBIAN_FRONTEND="noninteractive"
ARG WINBIND_PRIVILEGED_GID="998"

ENV RADIUS_LISTEN_ADDRESS="*" \
    RADIUS_LISTEN_PORT="1812" \
    RADIUS_CLIENT_REQUIRE_MESSAGE_AUTHENTICATOR="yes" \
    RADIUS_MAB_USERS_FILE="/run/radius-site/mab-users" \
    RADIUS_LDAP_PORT="389"

COPY clients.conf /etc/freeradius/3.0/clients.conf
COPY default inner-tunnel /etc/freeradius/3.0/sites-available/
COPY eap files mschap /etc/freeradius/3.0/mods-available/
COPY ldap /etc/freeradius/3.0/mods-enabled/ldap
COPY --chmod=755 entrypoint.sh /radius-site-entrypoint.sh

RUN apt-get update \
 && apt-get install -y --no-install-recommends winbind \
 && apt-get clean \
 && rm -rf /var/lib/apt/lists/* \
 && groupmod --gid "${WINBIND_PRIVILEGED_GID}" winbindd_priv \
 && usermod --append --groups winbindd_priv freerad \
 && mkdir -p \
      /run/radius-site \
      /run/secrets/radius-client \
      /run/secrets/radius-eap \
      /run/secrets/radius-ldap \
 && printf '%s\n' '# radius-site-mab-v1' > /run/radius-site/mab-users \
 && printf '%s\n' 'build-only-client' > /run/secrets/radius-client/secret \
 && printf '%s\n' 'build-only-password' > /run/secrets/radius-ldap/password \
 && cp /etc/ssl/private/ssl-cert-snakeoil.key /run/secrets/radius-eap/server.key \
 && cp /etc/ssl/certs/ssl-cert-snakeoil.pem /run/secrets/radius-eap/server.pem \
 && chown freerad:freerad /run/secrets/radius-eap/server.key \
 && chmod 0640 /run/secrets/radius-eap/server.key \
 && env \
      RADIUS_CLIENT_ADDRESS=192.0.2.0/24 \
      RADIUS_DEFAULT_VLAN=10 \
      RADIUS_EAP_VLAN_MAP=wifi-primary=10,wifi-secondary=20 \
      RADIUS_LDAP_BASE_DN=dc=example,dc=test \
      RADIUS_LDAP_IDENTITY=radius-reader@example.test \
      RADIUS_LDAP_SERVER=192.0.2.10 \
      RADIUS_WINBIND_DOMAIN=EXAMPLE \
      /radius-site-entrypoint.sh freeradius -XC \
 && rm -rf \
      /run/radius-site \
      /run/radius-site-eap-vlan \
      /run/secrets/radius-client \
      /run/secrets/radius-eap \
      /run/secrets/radius-ldap

EXPOSE 1812/udp

ENTRYPOINT ["/radius-site-entrypoint.sh"]
