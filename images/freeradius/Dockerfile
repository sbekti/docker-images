FROM ubuntu:24.04
ARG DEBIAN_FRONTEND=noninteractive
ARG FREERADIUS_VERSION=3.2.5*

#
#  We need curl to get the signing key
#
RUN apt-get update \
 && apt-get install -y curl \
 && apt-get clean \
 && rm -r /var/lib/apt/lists/*

#
#  Set up NetworkRADIUS extras repository
#
RUN install -d -o root -g root -m 0755 /etc/apt/keyrings \
 && curl -o /etc/apt/keyrings/packages.networkradius.com.asc "https://packages.inkbridgenetworks.com/pgp/packages%40networkradius.com" \
 && echo "deb [signed-by=/etc/apt/keyrings/packages.networkradius.com.asc] http://packages.networkradius.com/extras/ubuntu/jammy jammy main" > /etc/apt/sources.list.d/networkradius-extras.list

ARG freerad_uid=101
ARG freerad_gid=101

RUN groupadd -g ${freerad_gid} -r freerad \
 && useradd -u ${freerad_uid} -g freerad -r -M -d /etc/freeradius -s /usr/sbin/nologin freerad \
 && apt-get update \
 && apt-get install -y \
      freeradius=${FREERADIUS_VERSION}* \
      freeradius-ldap=${FREERADIUS_VERSION}* \
      freeradius-postgresql=${FREERADIUS_VERSION}* \
      freeradius-redis=${FREERADIUS_VERSION}* \
      freeradius-yubikey=${FREERADIUS_VERSION}* \
      jq \
 && apt-get clean \
 && rm -r /var/lib/apt/lists/* \
    \
 && ln -s /etc/freeradius /etc/raddb

WORKDIR /
COPY entrypoint.sh entrypoint.sh
RUN chmod +x entrypoint.sh

EXPOSE 1812/udp 1813/udp
ENTRYPOINT ["/entrypoint.sh"]
CMD ["freeradius"]
