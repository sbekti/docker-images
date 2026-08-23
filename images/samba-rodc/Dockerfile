FROM debian:13-slim

ARG SAMBA_MIN_VERSION="2:4.23.9"
ARG DEBIAN_FRONTEND="noninteractive"

RUN printf '%s\n' \
      'deb http://deb.debian.org/debian trixie-backports main' \
      > /etc/apt/sources.list.d/trixie-backports.list \
 && apt-get update \
 && apt-get install -y --no-install-recommends \
      ca-certificates \
 && apt-get install -y --no-install-recommends -t trixie-backports \
      krb5-user \
      samba-ad-dc \
      samba-ad-provision \
      samba-common-bin \
      winbind \
 && installed_version="$(dpkg-query -W -f='${Version}' samba)" \
 && if ! dpkg --compare-versions "${installed_version}" ge "${SAMBA_MIN_VERSION}"; then \
      echo "Samba ${installed_version} is older than required ${SAMBA_MIN_VERSION}." >&2; \
      exit 1; \
    fi \
 && apt-get clean \
 && rm -rf /var/lib/apt/lists/*

COPY --chmod=755 rodc-state-check.py /usr/local/sbin/rodc-state-check
COPY --chmod=755 entrypoint.sh /entrypoint.sh

RUN if /usr/local/sbin/rodc-state-check >/dev/null 2>&1; then \
      echo "RODC state validation unexpectedly accepted an empty image." >&2; \
      exit 1; \
    fi

VOLUME ["/var/lib/samba", "/etc/samba", "/run/samba"]

EXPOSE 135/tcp 389/tcp 389/udp 50000-50019/tcp

STOPSIGNAL SIGTERM

HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
    CMD ["/entrypoint.sh", "healthcheck"]

ENTRYPOINT ["/entrypoint.sh"]
CMD ["run"]
