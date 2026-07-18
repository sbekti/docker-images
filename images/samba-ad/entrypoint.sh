#!/bin/bash
set -euo pipefail

# Runtime defaults (override with environment variables)
: "${REALM:=EXAMPLE.COM}"
: "${DOMAIN:=EXAMPLE}"
: "${DNS_FORWARDER:=1.1.1.1 1.0.0.1}"
: "${RPC_PORT_START:=50000}"
: "${RPC_PORT_END:=50019}"
: "${DNS_UPDATE_MODE:=nonsecure and secure}"
: "${NETBIOS_NAME:=DC1}"
: "${EXTERNAL_IP:=127.0.0.1}"
: "${NTLM_AUTH:=no}"

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

if [[ -z "${ADMIN_PASS:-}" ]]; then
    echo "ERROR: ADMIN_PASS is required." >&2
    exit 1
fi

# Log resolved configuration
echo "=== Samba AD DC Configuration ==="
echo "  REALM:         ${REALM}"
echo "  DOMAIN:        ${DOMAIN}"
echo "  NETBIOS_NAME:  ${NETBIOS_NAME}"
echo "  EXTERNAL_IP:   ${EXTERNAL_IP}"
echo "  DNS_FORWARDER: ${DNS_FORWARDER}"
echo "  RPC_PORTS:     ${RPC_PORT_START}-${RPC_PORT_END}"
echo "  DNS_UPDATE:    ${DNS_UPDATE_MODE}"
echo "  NTLM_AUTH:     ${NTLM_AUTH}"
echo "  TLS_ENABLED:   ${TLS_ENABLED}"
echo "  TLS_CERTFILE:  ${TLS_CERTFILE}"
echo "  TLS_KEYFILE:   ${TLS_KEYFILE}"
echo "  TLS_CAFILE:    ${TLS_CAFILE:-<empty>}"
echo "================================="

# Keep lock directory permissions compatible with Samba's startup checks.
# /run/samba is created at runtime by package init scripts.
if [ -d /run/samba ]; then
    chmod 0755 /run/samba
fi

# Provision once; reuse persisted domain state on subsequent starts.
if [ -f /var/lib/samba/private/secrets.keytab ]; then
    echo "Domain already provisioned."
else
    echo "Provisioning domain..."
    rm -f /etc/samba/smb.conf
    
    # --host-ip forces the initial DNS A record to EXTERNAL_IP.
    samba-tool domain provision \
        --server-role=dc \
        --use-rfc2307 \
        --dns-backend=SAMBA_INTERNAL \
        --realm="${REALM}" \
        --domain="${DOMAIN}" \
        --host-name="${NETBIOS_NAME}" \
        --adminpass="${ADMIN_PASS}" \
        --host-ip="${EXTERNAL_IP}" \
        --option="dns forwarder = ${DNS_FORWARDER}" \
        --option="netbios name = ${NETBIOS_NAME}" \
        --option="rpc server port = ${RPC_PORT_START}-${RPC_PORT_END}" \
        --option="allow dns updates = ${DNS_UPDATE_MODE}" \
        --option="ntlm auth = ${NTLM_AUTH}" \
        --option="ldap server require strong auth = no" \
        --option="dns update command = /usr/bin/true"
    
    # Keep scheduled samba_dnsupdate from replacing the external DNS A record.
fi

if [[ ! -f /etc/samba/smb.conf ]]; then
    echo "ERROR: /etc/samba/smb.conf is missing after provisioning." >&2
    exit 1
fi

if [[ "${DNS_FORWARDER}" == *$'\n'* || "${DNS_FORWARDER}" == *$'\r'* ]]; then
    echo "ERROR: DNS_FORWARDER must be a single line." >&2
    exit 1
fi

# Reconcile the forwarder for both new and persisted domains.
sed -i -E '/^[[:space:]]*dns forwarder[[:space:]]*=/d' /etc/samba/smb.conf
sed -i "/^\\[global\\][[:space:]]*$/a\\\\tdns forwarder = ${DNS_FORWARDER}" /etc/samba/smb.conf

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

testparm -s /etc/samba/smb.conf >/dev/null
effective_netbios_name="$(testparm -s --parameter-name='netbios name' 2>/dev/null)"
if [[ "${effective_netbios_name,,}" != "${NETBIOS_NAME,,}" ]]; then
    echo "ERROR: smb.conf netbios name '${effective_netbios_name}' does not match NETBIOS_NAME '${NETBIOS_NAME}'." >&2
    exit 1
fi

# Prepare /etc/krb5.conf for in-container Kerberos admin tools (kinit, ldapsearch, etc.).
# The generated file prefers DNS KDC discovery, but cluster DNS often lacks Kerberos SRV records.
# Use explicit local KDC/admin_server entries in the existing realm block.
echo "Setting up /etc/krb5.conf for in-container Kerberos tools..."
if [ -f /var/lib/samba/private/krb5.conf ]; then
    cp /var/lib/samba/private/krb5.conf /etc/krb5.conf
    # Force static KDC lookup for this container.
    sed -i 's/dns_lookup_kdc = true/dns_lookup_kdc = false/g' /etc/krb5.conf
    # Insert local KDC/admin_server lines right after "REALM = {".
    sed -i "/^\s*${REALM} = {/a\\        kdc = 127.0.0.1\n        admin_server = 127.0.0.1" /etc/krb5.conf
fi

# Apply password policy settings
echo "Applying password policy..."
samba-tool domain passwordsettings set --complexity="${PWD_COMPLEXITY}"
samba-tool domain passwordsettings set --min-pwd-length="${PWD_MIN_LENGTH}"
samba-tool domain passwordsettings set --history-length="${PWD_HISTORY}"
samba-tool domain passwordsettings set --min-pwd-age="${PWD_MIN_AGE}"
samba-tool domain passwordsettings set --max-pwd-age="${PWD_MAX_AGE}"
samba-tool domain passwordsettings set --store-plaintext="${PWD_STORE_PLAINTEXT}"
echo "Password policy applied."

echo "Starting Samba AD DC..."
exec samba -i --no-process-group
