#!/bin/bash
set -euo pipefail

readonly RODC_SERVICES="rpc ldap cldap drepl winbindd kcc"

usage() {
    cat <<'EOF'
Usage: /entrypoint.sh <command>

Commands:
  run          Validate existing RODC state and start Samba
  join         Join lifecycle (implemented in milestone M02)
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

validate_rodc_state() {
    test -f /etc/samba/smb.conf \
        || fail "Missing /etc/samba/smb.conf; join this container as an RODC first."
    test -f /var/lib/samba/private/sam.ldb \
        || fail "Missing /var/lib/samba/private/sam.ldb; join this container as an RODC first."

    testparm --suppress-prompt -s /etc/samba/smb.conf >/dev/null
    /usr/local/sbin/rodc-state-check
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
        fail "The interactive RODC join is not available until milestone M02." 78
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
        usage
        ;;
    *)
        usage >&2
        fail "Unknown command '${command_name}'." 64
        ;;
esac
