# Keep the RouterOS-verified base; newer releases need a device smoke test.
ARG ALPINE_VERSION="3.24.1"
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
      --with-modules='rlm_always rlm_eap rlm_expr rlm_files rlm_ldap rlm_linelog rlm_mschap rlm_pap' \
 && make -j"$(nproc)" \
 && make install \
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
      rlm_linelog \
      rlm_mschap \
      rlm_pap; \
    do mv "/opt/lib/${library}.so" /opt/lib.keep/; done \
 && rm -rf /opt/lib \
 && mv /opt/lib.keep /opt/lib \
 && strip --strip-unneeded /opt/sbin/radiusd /opt/lib/*.so \
 && rm -rf /opt/etc/raddb/certs

FROM alpine:${ALPINE_VERSION}
RUN apk add --no-cache \
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

COPY --chmod=755 ntlm_auth_bridge /usr/local/bin/ntlm_auth_bridge
COPY --chmod=755 entrypoint.sh /radius-local-entrypoint.sh

ENV PATH="/opt/sbin:/opt/bin:${PATH}"
EXPOSE 1812/udp 18120/udp
ENTRYPOINT ["/radius-local-entrypoint.sh"]
CMD ["radiusd", "-f", "-l", "stdout", "-d", "/run/radius-local-runtime/config"]
