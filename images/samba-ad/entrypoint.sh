#!/bin/bash
set -e

# Default variables
: "${REALM:=EXAMPLE.COM}"
: "${DOMAIN:=EXAMPLE}"
: "${ADMIN_PASS:=Passw0rd}"
: "${DNS_FORWARDER:=8.8.8.8}"
: "${RPC_PORT_START:=50000}"
: "${RPC_PORT_END:=50019}"
: "${DNS_UPDATE_MODE:=nonsecure and secure}"
: "${NETBIOS_NAME:=DC1}"
: "${EXTERNAL_IP:=127.0.0.1}"

# Password policy defaults
: "${PWD_COMPLEXITY:=on}"
: "${PWD_MIN_LENGTH:=7}"
: "${PWD_HISTORY:=24}"
: "${PWD_MIN_AGE:=1}"
: "${PWD_MAX_AGE:=42}"
: "${PWD_STORE_PLAINTEXT:=off}"

# TLS defaults
: "${TLS_ENABLED:=no}"
: "${TLS_CERTFILE:=/etc/samba/tls/tls.crt}"
: "${TLS_KEYFILE:=/etc/samba/tls/tls.key}"
: "${TLS_CAFILE:=}"

# Log resolved configuration
echo "=== Samba AD DC Configuration ==="
echo "  REALM:         ${REALM}"
echo "  DOMAIN:        ${DOMAIN}"
echo "  NETBIOS_NAME:  ${NETBIOS_NAME}"
echo "  EXTERNAL_IP:   ${EXTERNAL_IP}"
echo "  DNS_FORWARDER: ${DNS_FORWARDER}"
echo "  RPC_PORTS:     ${RPC_PORT_START}-${RPC_PORT_END}"
echo "  DNS_UPDATE:    ${DNS_UPDATE_MODE}"
echo "  TLS_ENABLED:   ${TLS_ENABLED}"
echo "  TLS_CERTFILE:  ${TLS_CERTFILE}"
echo "  TLS_KEYFILE:   ${TLS_KEYFILE}"
echo "  TLS_CAFILE:    ${TLS_CAFILE:-<empty>}"
echo "================================="

# Set the system hostname to match the NetBIOS name
echo "Setting system hostname to ${NETBIOS_NAME}..."
hostname "${NETBIOS_NAME}"
if [ "$(hostname)" != "${NETBIOS_NAME}" ]; then
    echo "WARNING: Failed to set hostname to ${NETBIOS_NAME}. Running as $(hostname)." >&2
fi

# Fix testparm warning about lock directory permissions
# /run/samba is created at runtime by samba package init scripts
if [ -d /run/samba ]; then
    chmod 0755 /run/samba
fi

# Clean up /etc/hosts to remove Docker's internal IP entry for the hostname
# This ensures local resolution uses the External IP we inject
if ! grep -q "^${EXTERNAL_IP}.*${NETBIOS_NAME}" /etc/hosts; then
    echo "Patching /etc/hosts..."
    # Read existing hosts, excluding lines ending with our hostname (internal IP mappings)
    EXISTING_HOSTS=$(grep -v "[[:space:]]${NETBIOS_NAME}$" /etc/hosts)
    
    # Prepend our External IP mapping and overwrite the file
    echo "${EXTERNAL_IP} ${NETBIOS_NAME}.${REALM} ${NETBIOS_NAME}"$'\n'"${EXISTING_HOSTS}" > /etc/hosts
fi

# Check if domain is already provisioned
if [ -f /var/lib/samba/private/secrets.keytab ]; then
    echo "Domain already provisioned."
else
    echo "Provisioning domain..."
    rm -f /etc/samba/smb.conf
    
    # Run provisioning
    # --host-ip: Forces the initial DNS A record to the External IP
    samba-tool domain provision \
        --server-role=dc \
        --use-rfc2307 \
        --dns-backend=SAMBA_INTERNAL \
        --realm="${REALM}" \
        --domain="${DOMAIN}" \
        --adminpass="${ADMIN_PASS}" \
        --host-ip="${EXTERNAL_IP}" \
        --option="dns forwarder = ${DNS_FORWARDER}" \
        --option="netbios name = ${NETBIOS_NAME}" \
        --option="rpc server port = ${RPC_PORT_START}-${RPC_PORT_END}" \
        --option="allow dns updates = ${DNS_UPDATE_MODE}" \
        --option="ldap server require strong auth = no" \
        --option="dns update command = /usr/bin/true"
    
    # "dns update command = /usr/bin/true" prevents samba_dnsupdate from 
    # overwriting our External IP with the Pod IP on scheduled runs.
fi

# Configure TLS in smb.conf (runs every start to ensure settings are always current)
if [ "${TLS_ENABLED}" = "yes" ]; then
    echo "Configuring TLS in smb.conf..."
    # Remove any existing TLS lines to avoid duplicates on restart
    sed -i '/^\s*tls enabled\s*=/d' /etc/samba/smb.conf
    sed -i '/^\s*tls certfile\s*=/d' /etc/samba/smb.conf
    sed -i '/^\s*tls keyfile\s*=/d' /etc/samba/smb.conf
    sed -i '/^\s*tls cafile\s*=/d' /etc/samba/smb.conf
    # Inject TLS settings into [global] section right after the [global] line
    sed -i "/^\[global\]/a\\\\ttls cafile = ${TLS_CAFILE}\\n\\ttls keyfile = ${TLS_KEYFILE}\\n\\ttls certfile = ${TLS_CERTFILE}\\n\\ttls enabled = yes" /etc/samba/smb.conf
    echo "TLS configured: certfile=${TLS_CERTFILE}, keyfile=${TLS_KEYFILE}, cafile=${TLS_CAFILE:-<empty>}"
fi

# Set up Kerberos for local debugging
# samba-tool generates /var/lib/samba/private/krb5.conf with dns_lookup_kdc = true,
# which breaks kinit inside the pod because K8s CoreDNS cannot resolve Kerberos SRV records.
# Fix: copy the generated config, disable DNS-based KDC discovery, and inject explicit
# kdc/admin_server entries into the existing [realms] block (which already has default_domain).
# We sed-insert into the existing block to avoid creating a duplicate [realms] section.
echo "Setting up /etc/krb5.conf for local debugging..."
if [ -f /var/lib/samba/private/krb5.conf ]; then
    cp /var/lib/samba/private/krb5.conf /etc/krb5.conf
    # Disable DNS-based KDC discovery (K8s CoreDNS can't resolve _kerberos._tcp SRV records)
    sed -i 's/dns_lookup_kdc = true/dns_lookup_kdc = false/g' /etc/krb5.conf
    # Insert explicit KDC address into the existing [realms] block right after "REALM = {"
    # The generated krb5.conf already has: REALM = {\n\t\tdefault_domain = ...\n\t}
    # We add kdc and admin_server before the existing entries.
    sed -i "/^\s*${REALM} = {/a\\        kdc = 127.0.0.1\n        admin_server = 127.0.0.1" /etc/krb5.conf
fi

# Apply password policy settings
echo "Applying password policy..."
samba-tool domain passwordsettings set --complexity="${PWD_COMPLEXITY}" 2>/dev/null || true
samba-tool domain passwordsettings set --min-pwd-length="${PWD_MIN_LENGTH}" 2>/dev/null || true
samba-tool domain passwordsettings set --history-length="${PWD_HISTORY}" 2>/dev/null || true
samba-tool domain passwordsettings set --min-pwd-age="${PWD_MIN_AGE}" 2>/dev/null || true
samba-tool domain passwordsettings set --max-pwd-age="${PWD_MAX_AGE}" 2>/dev/null || true
samba-tool domain passwordsettings set --store-plaintext="${PWD_STORE_PLAINTEXT}" 2>/dev/null || true
echo "Password policy applied."

echo "Starting Samba AD DC..."
exec samba -i --no-process-group
