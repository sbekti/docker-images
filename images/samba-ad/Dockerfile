FROM ubuntu:24.04

# Avoid interactive prompts during build
ENV DEBIAN_FRONTEND=noninteractive

# Install Samba and related tools
# dnsutils: for testing DNS resolution (dig/nslookup)
# iproute2: for network interface inspection (ip/ss)
RUN apt-get update && apt-get install -y \
    samba-ad-dc \
    krb5-user \
    dnsutils \
    iproute2 \
    iputils-ping \
    ca-certificates \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Expose necessary ports
# DNS: 53 TCP/UDP
# Kerberos: 88 TCP/UDP
# RPC endpoint mapper: 135 TCP
# NetBIOS Name Service: 137 UDP
# NetBIOS Datagram Service: 138 UDP
# NetBIOS Session Service: 139 TCP
# LDAP: 389 TCP/UDP
# SMB: 445 TCP
# Kerberos password change: 464 TCP/UDP
# LDAPS: 636 TCP
# Global Catalog: 3268 TCP
# Global Catalog SSL: 3269 TCP
# RPC default endpoint: 49152 TCP
EXPOSE 53/tcp 53/udp 88/tcp 88/udp 135/tcp 137/udp 138/udp 139/tcp 389/tcp 389/udp 445/tcp 464/tcp 464/udp 636/tcp 3268/tcp 3269/tcp 49152/tcp

# Setup directories for persistent data
VOLUME ["/var/lib/samba", "/etc/samba", "/run/samba"]

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
