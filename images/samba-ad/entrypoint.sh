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
: "${DNS_HOSTNAME:=${NETBIOS_NAME,,}.${REALM,,}}"
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

# Log resolved configuration
echo "=== Samba AD DC Configuration ==="
echo "  REALM:         ${REALM}"
echo "  DOMAIN:        ${DOMAIN}"
echo "  NETBIOS_NAME:  ${NETBIOS_NAME}"
echo "  DNS_HOSTNAME:  ${DNS_HOSTNAME}"
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
    if [[ -z "${ADMIN_PASS:-}" ]]; then
        echo "ERROR: ADMIN_PASS is required to provision a domain." >&2
        exit 1
    fi

    echo "Provisioning domain..."
    rm -f /etc/samba/smb.conf

    # --host-ip forces the initial DNS A record to EXTERNAL_IP.
    samba-tool domain provision \
        --server-role=dc \
        --use-rfc2307 \
        --dns-backend=SAMBA_INTERNAL \
        --realm="${REALM}" \
        --domain="${DOMAIN}" \
        --host-name="${DNS_HOSTNAME%%.*}" \
        --adminpass="${ADMIN_PASS}" \
        --host-ip="${EXTERNAL_IP}" \
        --option="netbios name = ${NETBIOS_NAME}" \
        --option="dns hostname = ${DNS_HOSTNAME}"
fi

if [[ ! -f /etc/samba/smb.conf ]]; then
    echo "ERROR: /etc/samba/smb.conf is missing after provisioning." >&2
    exit 1
fi

for name in DNS_FORWARDER RPC_PORT_START RPC_PORT_END DNS_UPDATE_MODE DNS_HOSTNAME NTLM_AUTH TLS_ENABLED TLS_CERTFILE TLS_KEYFILE TLS_CAFILE; do
    value=${!name}
    if [[ "${value}" == *$'\n'* || "${value}" == *$'\r'* ]]; then
        echo "ERROR: ${name} must be a single line." >&2
        exit 1
    fi
done

case "${TLS_ENABLED}" in
    yes | no) ;;
    *) echo "ERROR: TLS_ENABLED must be yes or no." >&2; exit 1 ;;
esac

# Mutable settings have one owner for both new and persisted domains.
readonly runtime_conf=/etc/samba/runtime.conf
sed -i -E \
    -e '/^[[:space:]]*(dns forwarder|allow dns updates|rpc server port|rpc server dynamic port range|ntlm auth|ldap server require strong auth|dns update command|tls enabled|tls certfile|tls keyfile|tls cafile)[[:space:]]*=/Id' \
    -e '/^[[:space:]]*include[[:space:]]*=[[:space:]]*\/etc\/samba\/runtime\.conf[[:space:]]*$/Id' \
    /etc/samba/smb.conf
sed -i "/^\[global\][[:space:]]*$/a\\\tinclude = ${runtime_conf}" /etc/samba/smb.conf

{
    printf '\tdns forwarder = %s\n' "${DNS_FORWARDER}"
    printf '\tallow dns updates = %s\n' "${DNS_UPDATE_MODE}"
    printf '\trpc server dynamic port range = %s-%s\n' "${RPC_PORT_START}" "${RPC_PORT_END}"
    printf '\tntlm auth = %s\n' "${NTLM_AUTH}"
    printf '\tldap server require strong auth = no\n'
    printf '\tdns update command = /usr/bin/true\n'
    printf '\ttls enabled = %s\n' "${TLS_ENABLED}"
    if [[ "${TLS_ENABLED}" == yes ]]; then
        printf '\ttls certfile = %s\n' "${TLS_CERTFILE}"
        printf '\ttls keyfile = %s\n' "${TLS_KEYFILE}"
        printf '\ttls cafile = %s\n' "${TLS_CAFILE}"
    fi
} > "${runtime_conf}"
chmod 0644 "${runtime_conf}"

testparm -s /etc/samba/smb.conf >/dev/null
check_identity() {
    local parameter=$1 expected=$2 actual
    actual="$(testparm -s --parameter-name="${parameter}" 2>/dev/null)"
    if [[ "${actual,,}" != "${expected,,}" ]]; then
        echo "ERROR: smb.conf ${parameter} '${actual}' does not match '${expected}'." >&2
        exit 1
    fi
}
check_identity "netbios name" "${NETBIOS_NAME}"
check_identity realm "${REALM}"
check_identity workgroup "${DOMAIN}"
check_identity "dns hostname" "${DNS_HOSTNAME}"

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
samba-tool domain passwordsettings set \
    --complexity="${PWD_COMPLEXITY}" \
    --min-pwd-length="${PWD_MIN_LENGTH}" \
    --history-length="${PWD_HISTORY}" \
    --min-pwd-age="${PWD_MIN_AGE}" \
    --max-pwd-age="${PWD_MAX_AGE}" \
    --store-plaintext="${PWD_STORE_PLAINTEXT}"
echo "Password policy applied."

echo "Starting Samba AD DC..."
exec samba -i --no-process-group
