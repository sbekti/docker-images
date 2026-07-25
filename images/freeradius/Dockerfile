FROM ubuntu:26.04
ARG DEBIAN_FRONTEND=noninteractive

#
#  We need curl to get the signing key
#
RUN apt-get update \
 && apt-get install -y curl \
 && apt-get clean \
 && rm -r /var/lib/apt/lists/*

#
#  Set up NetworkRADIUS extras repository (libkqueue dependency)
#
RUN install -d -o root -g root -m 0755 /etc/apt/keyrings \
 && curl -o /etc/apt/keyrings/packages.networkradius.com.asc "https://packages.inkbridgenetworks.com/pgp/packages%40networkradius.com" \
 && echo "deb [signed-by=/etc/apt/keyrings/packages.networkradius.com.asc] http://packages.networkradius.com/extras/ubuntu/noble noble main" > /etc/apt/sources.list.d/networkradius-extras.list

ARG freerad_uid=101
ARG freerad_gid=101

RUN groupadd -g ${freerad_gid} -r freerad \
 && useradd -u ${freerad_uid} -g freerad -r -M -d /etc/freeradius -s /usr/sbin/nologin freerad \
 && apt-get update \
 && apt-get install -y \
      freeradius \
      freeradius-ldap \
      freeradius-postgresql \
      freeradius-redis \
      freeradius-yubikey \
      jq \
 && apt-get clean \
 && rm -r /var/lib/apt/lists/* \
    \
 && ln -s /etc/freeradius /etc/raddb

WORKDIR /
COPY --chmod=755 entrypoint.sh entrypoint.sh

EXPOSE 1812/udp 1813/udp
ENTRYPOINT ["/entrypoint.sh"]
CMD ["freeradius"]
