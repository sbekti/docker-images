#!/bin/bash
set -euo pipefail

readonly RODC_SERVICES="rpc, ldap, cldap, drepl, winbindd"
readonly SMB_CONF="/etc/samba/smb.conf"
readonly SAM_DB="/var/lib/samba/private/sam.ldb"
readonly RPC_PORT_RANGE="50000-50019"
readonly SAMBA_RUNTIME_DIR="/run/samba"
readonly WINBIND_PRIVILEGED_DIR="/var/lib/samba/winbindd_privileged"
readonly SAMBA_PREFORK_CHILDREN="${SAMBA_PREFORK_CHILDREN:-1}"

usage() {
    cat <<'EOF'
Usage: /entrypoint.sh <command>

Commands:
  run          Validate existing RODC state and start Samba
  join         Interactively join a fresh volume as an RODC
  healthcheck  Ping local Samba and Winbind
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

set_runtime_configuration() {
    sed -i -E \
        '/^[[:space:]]*(dns update command|ldap server require strong auth|ntlm auth|rpc server dynamic port range|server services)[[:space:]]*=/Id' \
        "${SMB_CONF}"
    sed -i "/^\[global\][[:space:]]*$/a\\
\tdns update command = /usr/bin/false\n\
\tldap server require strong auth = no\n\
\tntlm auth = mschapv2-and-ntlmv2-only\n\
\trpc server dynamic port range = ${RPC_PORT_RANGE}\n\
\tserver services = ${RODC_SERVICES}" "${SMB_CONF}"
}

validate_rodc_state() {
    test -f "${SMB_CONF}" \
        || fail "Missing ${SMB_CONF}; join this container as an RODC first."
    test -f "${SAM_DB}" \
        || fail "Missing ${SAM_DB}; join this container as an RODC first."

    testparm --suppress-prompt -s "${SMB_CONF}" >/dev/null
    /usr/local/sbin/rodc-state-check
}

validate_runtime_configuration() {
    [[ "${SAMBA_PREFORK_CHILDREN}" =~ ^[1-9][0-9]*$ ]] \
        || fail "SAMBA_PREFORK_CHILDREN must be a positive integer."
}

prepare_runtime() {
    install -d -m 0755 -o root -g root "${SAMBA_RUNTIME_DIR}"
    install -d -m 0750 -o root -g winbindd_priv "${WINBIND_PRIVILEGED_DIR}"
    find "${SAMBA_RUNTIME_DIR}" "${WINBIND_PRIVILEGED_DIR}" \
        -mindepth 1 -delete
}

join_rodc() {
    local dns_hostname

    require_variable REALM
    require_variable NETBIOS_NAME
    require_variable RODC_ADDRESS
    require_variable RWDC_ADDRESS
    require_variable JOIN_USER

    [[ -t 0 && -t 1 ]] \
        || fail "join requires an interactive terminal so Samba can prompt for the password."
    [[ ! -e "${SMB_CONF}" ]] \
        || fail "${SMB_CONF} already exists; join requires fresh /etc/samba state."
    [[ ! -e "${SAM_DB}" ]] \
        || fail "${SAM_DB} already exists; join requires fresh /var/lib/samba state."

    dns_hostname="${NETBIOS_NAME,,}.${REALM,,}"
    umask 077
    samba-tool domain join "${REALM,,}" RODC \
        --server="${RWDC_ADDRESS}" \
        --realm="${REALM}" \
        --username="${JOIN_USER}" \
        --dns-backend=NONE \
        --no-dns-updates \
        --option="netbios name = ${NETBIOS_NAME}" \
        --option="dns hostname = ${dns_hostname}" \
        --option="interfaces = lo ${RODC_ADDRESS}" \
        --option="bind interfaces only = yes" \
        --option="dns update command = /usr/bin/false" \
        --option="ldap server require strong auth = no" \
        --option="ntlm auth = mschapv2-and-ntlmv2-only" \
        --option="rpc server dynamic port range = ${RPC_PORT_RANGE}" \
        --option="server services = ${RODC_SERVICES}"

    set_runtime_configuration
    validate_rodc_state
    echo "RODC join completed."
}

command_name="${1:-run}"
shift || true

case "${command_name}" in
    run)
        require_no_arguments run "$@"
        validate_runtime_configuration
        validate_rodc_state
        prepare_runtime
        exec samba \
            --foreground \
            --no-process-group \
            --option="prefork children = ${SAMBA_PREFORK_CHILDREN}" \
            --option="server services = ${RODC_SERVICES}"
        ;;
    join)
        require_no_arguments join "$@"
        join_rodc
        ;;
    healthcheck)
        require_no_arguments healthcheck "$@"
        smbcontrol samba ping >/dev/null
        wbinfo --ping >/dev/null
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
