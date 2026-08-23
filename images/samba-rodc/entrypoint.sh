#!/bin/bash
set -euo pipefail

readonly RODC_SERVICES="rpc, ldap, cldap, drepl, winbindd, kcc"
readonly SMB_CONF="/etc/samba/smb.conf"
readonly SAM_DB="/var/lib/samba/private/sam.ldb"

export RPC_PORT_START="${RPC_PORT_START:-50000}"
export RPC_PORT_END="${RPC_PORT_END:-50019}"

usage() {
    cat <<'EOF'
Usage: /entrypoint.sh <command>

Commands:
  run          Validate existing RODC state and start Samba
  join         Interactively join a fresh volume as an RODC
  sync-once    Cache reconciliation (implemented in milestone M03)
  healthcheck  Validate RODC state and confirm the Samba process responds
EOF
}

fail() {
    local message="$1"
    local status="${2:-1}"

    echo "ERROR: ${message}" >&2
    exit "${status}"
}

require_no_arguments() {
    local command_name="$1"
    shift

    if (( $# != 0 )); then
        fail "${command_name} does not accept arguments."
    fi
}

require_variable() {
    local variable_name="$1"

    if [[ -z "${!variable_name:-}" ]]; then
        fail "${variable_name} is required."
    fi
}

validate_ipv4() {
    local variable_name="$1"

    python3 - "${!variable_name}" <<'PY' || fail "${variable_name} must be a literal IPv4 address."
import ipaddress
import sys

address = ipaddress.ip_address(sys.argv[1])
if address.version != 4:
    raise ValueError("not IPv4")
PY
}

validate_configuration() {
    local expected_dns_hostname

    require_variable REALM
    require_variable DOMAIN
    require_variable NETBIOS_NAME
    require_variable DNS_HOSTNAME
    require_variable RODC_ADDRESS

    [[ "${REALM}" =~ ^[A-Z0-9]([A-Z0-9.-]*[A-Z0-9])?$ && "${REALM}" == *.* ]] \
        || fail "REALM must be an uppercase DNS domain."
    [[ "${DOMAIN}" =~ ^[A-Z0-9][A-Z0-9_-]{0,14}$ ]] \
        || fail "DOMAIN must be an uppercase NetBIOS name of at most 15 characters."
    [[ "${NETBIOS_NAME}" =~ ^[A-Z0-9][A-Z0-9_-]{0,14}$ ]] \
        || fail "NETBIOS_NAME must be an uppercase NetBIOS name of at most 15 characters."

    expected_dns_hostname="${NETBIOS_NAME,,}.${REALM,,}"
    [[ "${DNS_HOSTNAME}" == "${expected_dns_hostname}" ]] \
        || fail "DNS_HOSTNAME must be ${expected_dns_hostname}."

    validate_ipv4 RODC_ADDRESS

    [[ "${RPC_PORT_START}" =~ ^[0-9]+$ && "${RPC_PORT_END}" =~ ^[0-9]+$ ]] \
        || fail "RPC port bounds must be integers."
    (( RPC_PORT_START >= 1024 && RPC_PORT_END <= 65535 && RPC_PORT_START <= RPC_PORT_END )) \
        || fail "RPC port range must be ordered and between 1024 and 65535."
}

reconcile_smb_configuration() {
    sed -i -E \
        '/^[[:space:]]*(dns update command|ldap server require strong auth|ntlm auth|rpc server dynamic port range|server services)[[:space:]]*=/Id' \
        "${SMB_CONF}"
    sed -i "/^\[global\][[:space:]]*$/a\\
\tdns update command = /usr/bin/false\n\
\tldap server require strong auth = yes\n\
\tntlm auth = mschapv2-and-ntlmv2-only\n\
\trpc server dynamic port range = ${RPC_PORT_START}-${RPC_PORT_END}\n\
\tserver services = ${RODC_SERVICES}" "${SMB_CONF}"
}

validate_rodc_state() {
    validate_configuration
    test -f "${SMB_CONF}" \
        || fail "Missing ${SMB_CONF}; join this container as an RODC first."
    test -f "${SAM_DB}" \
        || fail "Missing ${SAM_DB}; join this container as an RODC first."

    testparm --suppress-prompt -s "${SMB_CONF}" >/dev/null
    /usr/local/sbin/rodc-state-check
}

join_rodc() {
    for secret_variable in ADMIN_PASS JOIN_PASSWORD SAMBA_PASSWORD; do
        if [[ -n "${!secret_variable:-}" ]]; then
            fail "Do not pass passwords through ${secret_variable}; join prompts on the terminal."
        fi
    done

    validate_configuration
    require_variable RWDC_ADDRESS
    require_variable JOIN_USER
    validate_ipv4 RWDC_ADDRESS
    [[ "${JOIN_USER}" =~ ^[A-Za-z0-9._-]+$ ]] \
        || fail "JOIN_USER must be a plain AD account name without a domain or password."

    [[ -t 0 && -t 1 ]] \
        || fail "join requires an interactive terminal so Samba can prompt for the password."
    [[ ! -e "${SMB_CONF}" ]] \
        || fail "${SMB_CONF} already exists; join requires fresh /etc/samba state."
    [[ ! -e "${SAM_DB}" ]] \
        || fail "${SAM_DB} already exists; join requires fresh /var/lib/samba state."

    umask 077
    samba-tool domain join "${REALM,,}" RODC \
        --server="${RWDC_ADDRESS}" \
        --realm="${REALM}" \
        --username="${JOIN_USER}" \
        --dns-backend=NONE \
        --no-dns-updates \
        --option="netbios name = ${NETBIOS_NAME}" \
        --option="dns hostname = ${DNS_HOSTNAME}" \
        --option="interfaces = lo ${RODC_ADDRESS}" \
        --option="bind interfaces only = yes" \
        --option="dns update command = /usr/bin/false" \
        --option="ldap server require strong auth = yes" \
        --option="ntlm auth = mschapv2-and-ntlmv2-only" \
        --option="rpc server dynamic port range = ${RPC_PORT_START}-${RPC_PORT_END}" \
        --option="server services = ${RODC_SERVICES}"

    reconcile_smb_configuration
    validate_rodc_state
    echo "Joined ${DNS_HOSTNAME} as a Samba RODC."
}

command_name="${1:-run}"
shift || true

case "${command_name}" in
    run)
        require_no_arguments run "$@"
        validate_rodc_state
        exec samba \
            --foreground \
            --no-process-group \
            --option="server services = ${RODC_SERVICES}"
        ;;
    join)
        require_no_arguments join "$@"
        join_rodc
        ;;
    sync-once)
        require_no_arguments sync-once "$@"
        fail "Credential-cache reconciliation is not available until milestone M03." 78
        ;;
    healthcheck)
        require_no_arguments healthcheck "$@"
        validate_rodc_state
        smbcontrol samba ping >/dev/null
        ;;
    help|-h|--help)
        require_no_arguments help "$@"
        usage
        ;;
    *)
        usage >&2
        fail "Unknown command '${command_name}'." 64
        ;;
esac
