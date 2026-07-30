FROM ubuntu:26.04
ARG DEBIAN_FRONTEND=noninteractive

ARG freerad_uid=101
ARG freerad_gid=101

RUN groupadd -g ${freerad_gid} -r freerad \
 && useradd -u ${freerad_uid} -g freerad -r -M -d /etc/freeradius -s /usr/sbin/nologin freerad \
 && apt-get update \
 && apt-get install -y \
      curl \
      freeradius \
      freeradius-ldap \
      freeradius-postgresql \
      freeradius-redis \
      freeradius-yubikey \
      jq \
 && apt-get clean \
 && rm -rf /var/lib/apt/lists/* \
    \
 && ln -s /etc/freeradius /etc/raddb

WORKDIR /
COPY --chmod=755 entrypoint.sh entrypoint.sh

EXPOSE 1812/udp 1813/udp
ENTRYPOINT ["/entrypoint.sh"]
CMD ["freeradius"]
