ARG ALPINE_VERSION="3.24"
ARG FREERADIUS_RELEASE="release_3_2_10"
ARG FREERADIUS_SHA256="3e9f24439ce976c04e4c56441722a3bd18677a61dd17f67bc8863eb4ed36cd4b"

FROM alpine:${ALPINE_VERSION} AS source
ARG FREERADIUS_RELEASE
ARG FREERADIUS_SHA256
RUN apk add --no-cache curl \
 && curl -fsSL -o /tmp/freeradius.tar.gz \
      "https://github.com/FreeRADIUS/freeradius-server/archive/refs/tags/${FREERADIUS_RELEASE}.tar.gz" \
 && echo "${FREERADIUS_SHA256}  /tmp/freeradius.tar.gz" | sha256sum -c - \
 && mkdir /src \
 && tar -xzf /tmp/freeradius.tar.gz --strip-components=1 -C /src

FROM alpine:${ALPINE_VERSION} AS build
RUN apk add --no-cache \
      build-base \
      linux-headers \
      openldap-dev \
      openssl \
      openssl-dev \
      pcre-dev \
      talloc-dev
COPY --from=source /src /src
WORKDIR /src
RUN : > src/modules/stable \
 && ./configure \
      --prefix=/opt \
      --with-modules='rlm_always rlm_eap rlm_expr rlm_files rlm_ldap rlm_mschap rlm_pap' \
 && make -j"$(nproc)" \
 && make install \
 && openssl pkey -in /opt/etc/raddb/certs/server.key \
      -passin pass:whatever \
      -out /opt/etc/raddb/certs/server-unencrypted.key \
 && find /opt/bin -type f ! -name radclient -delete \
 && rm -rf /opt/include /opt/share/doc /opt/share/man \
      /opt/sbin/checkrad /opt/sbin/raddebug /opt/sbin/radmin \
      /opt/sbin/rc.radiusd \
 && mkdir /opt/lib.keep \
 && for library in \
      libfreeradius-eap \
      libfreeradius-radius \
      libfreeradius-server \
      rlm_always \
      rlm_eap \
      rlm_eap_mschapv2 \
      rlm_eap_peap \
      rlm_eap_tls \
      rlm_expr \
      rlm_files \
      rlm_ldap \
      rlm_mschap \
      rlm_pap; \
    do mv "/opt/lib/${library}.so" /opt/lib.keep/; done \
 && rm -rf /opt/lib \
 && mv /opt/lib.keep /opt/lib \
 && strip --strip-unneeded /opt/sbin/radiusd /opt/lib/*.so

FROM alpine:${ALPINE_VERSION}
RUN apk add --no-cache \
      coreutils \
      curl \
      jq \
      libcrypto3 \
      libldap \
      libssl3 \
      pcre \
      talloc \
      tzdata \
 && addgroup -S radius \
 && adduser -S -D -H -G radius radius

COPY --from=build /opt /opt

RUN rm -f /opt/etc/raddb/mods-enabled/* /opt/etc/raddb/sites-enabled/* \
 && cp /opt/etc/raddb/mods-available/always \
      /opt/etc/raddb/mods-available/expr \
      /opt/etc/raddb/mods-available/pap \
      /opt/etc/raddb/mods-enabled/ \
 && sed -i \
      -e 's/^#[[:space:]]*user = radius/user = radius/' \
      -e 's/^#[[:space:]]*group = radius/group = radius/' \
      -e 's/^[[:space:]]*destination = files/destination = stdout/' \
      /opt/etc/raddb/radiusd.conf \
 && chown -R radius:radius /opt/var

COPY clients.conf /opt/etc/raddb/clients.conf
COPY default inner-tunnel /opt/etc/raddb/sites-enabled/
COPY eap files ldap mschap /opt/etc/raddb/mods-enabled/
COPY --chmod=755 ntlm_auth_bridge /usr/local/bin/ntlm_auth_bridge
COPY --chmod=755 entrypoint.sh /radius-local-entrypoint.sh

ENV PATH="/opt/sbin:/opt/bin:${PATH}" \
    RADIUS_LISTEN_ADDRESS="*" \
    RADIUS_LISTEN_PORT="1812" \
    RADIUS_CLIENT_REQUIRE_MESSAGE_AUTHENTICATOR="yes" \
    RADIUS_MAB_USERS_FILE="/run/radius-local/mab-users" \
    RADIUS_LDAP_PORT="389"

RUN chgrp -R radius /opt/etc/raddb \
 && chmod -R g+rX /opt/etc/raddb \
 && mkdir -p \
      /opt/etc/raddb/mods-config/files/mab_users \
      /run/radius-local \
      /run/secrets/radius-client \
      /run/secrets/radius-eap \
      /run/secrets/radius-ldap \
 && printf '%s\n' '# radius-mab-v1' > /run/radius-local/mab-users \
 && printf '%s\n' 'build-only-client' > /run/secrets/radius-client/secret \
 && printf '%s\n' 'build-only-password' > /run/secrets/radius-ldap/password \
 && cp /opt/etc/raddb/certs/server-unencrypted.key /run/secrets/radius-eap/server.key \
 && cp /opt/etc/raddb/certs/server.pem /run/secrets/radius-eap/server.pem \
 && env \
      RADIUS_CLIENT_ADDRESS=192.0.2.0/24 \
      RADIUS_DEFAULT_VLAN=10 \
      RADIUS_EAP_VLAN_MAP=wifi-primary=10,wifi-secondary=20 \
      RADIUS_LDAP_BASE_DN=dc=example,dc=test \
      RADIUS_LDAP_IDENTITY=radius-reader@example.test \
      RADIUS_LDAP_SERVER=192.0.2.10 \
      RADIUS_NTLM_AUTH_BRIDGE_URL=http://192.0.2.11:9555 \
      RADIUS_WINBIND_DOMAIN=EXAMPLE \
      /radius-local-entrypoint.sh radiusd -XC \
 && rm -rf \
      /opt/etc/raddb/mods-config/files/mab_users/authorize \
      /opt/etc/raddb/certs \
      /run/radius-local \
      /run/radius-local-eap-vlan \
      /run/secrets/radius-client \
      /run/secrets/radius-eap \
      /run/secrets/radius-ldap

EXPOSE 1812/udp

ENTRYPOINT ["/radius-local-entrypoint.sh"]
CMD ["radiusd"]
