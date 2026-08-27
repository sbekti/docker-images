#!/bin/sh
set -eu

fail() {
    echo "ERROR: $1" >&2
    exit 1
}

require_file() {
    [ -s "$1" ] || fail "Missing or empty $1."
}

valid_vlan() {
    case "$1" in
        ''|*[!0-9]*) return 1 ;;
    esac
    [ "$1" -ge 1 ] && [ "$1" -le 4094 ]
}

render_eap_vlan_map() (
    IFS=,
    : > /run/radius-site-eap-vlan

    for mapping in ${RADIUS_EAP_VLAN_MAP}; do
        group="${mapping%%=*}"
        vlan="${mapping#*=}"

        [ -n "${group}" ] && [ "${vlan}" != "${mapping}" ] \
            && [ "${vlan#*=}" = "${vlan}" ] \
            || fail "RADIUS_EAP_VLAN_MAP must contain group=vlan entries."
        valid_vlan "${vlan}" || fail "Invalid VLAN in RADIUS_EAP_VLAN_MAP."

        escaped_group="$(printf '%s' "${group}" | sed 's/\\/\\\\/g; s/"/\\"/g')"
        printf '%s\n' \
            "DEFAULT LDAP-Group == \"${escaped_group}\"" \
            "    Tunnel-Type := VLAN," \
            "    Tunnel-Medium-Type := IEEE-802," \
            "    Tunnel-Private-Group-Id := \"${vlan}\"" \
            >> /run/radius-site-eap-vlan
    done

    chmod 0644 /run/radius-site-eap-vlan
)

: "${RADIUS_CLIENT_ADDRESS:?RADIUS_CLIENT_ADDRESS is required.}"
: "${RADIUS_DEFAULT_VLAN:?RADIUS_DEFAULT_VLAN is required.}"
: "${RADIUS_EAP_VLAN_MAP:?RADIUS_EAP_VLAN_MAP is required.}"
: "${RADIUS_LDAP_BASE_DN:?RADIUS_LDAP_BASE_DN is required.}"
: "${RADIUS_LDAP_IDENTITY:?RADIUS_LDAP_IDENTITY is required.}"
: "${RADIUS_LDAP_SERVER:?RADIUS_LDAP_SERVER is required.}"
: "${RADIUS_NTLM_AUTH_BRIDGE_URL:?RADIUS_NTLM_AUTH_BRIDGE_URL is required.}"
: "${RADIUS_WINBIND_DOMAIN:?RADIUS_WINBIND_DOMAIN is required.}"

valid_vlan "${RADIUS_DEFAULT_VLAN}" || fail "RADIUS_DEFAULT_VLAN must be 1-4094."

require_file /run/secrets/radius-client/secret
require_file /run/secrets/radius-eap/server.key
require_file /run/secrets/radius-eap/server.pem
require_file /run/secrets/radius-ldap/password
require_file "${RADIUS_MAB_USERS_FILE}"
IFS= read -r marker < "${RADIUS_MAB_USERS_FILE}" || true
[ "${marker:-}" = "# radius-site-mab-v1" ] \
    || fail "Invalid MAB snapshot marker in ${RADIUS_MAB_USERS_FILE}."
install -d -m 0750 -o freerad -g freerad /run/radius-site-eap
install -m 0640 -o freerad -g freerad /run/secrets/radius-eap/server.key \
    /run/radius-site-eap/server.key
install -m 0644 -o freerad -g freerad /run/secrets/radius-eap/server.pem \
    /run/radius-site-eap/server.pem
install -m 0644 "${RADIUS_MAB_USERS_FILE}" \
    /etc/freeradius/3.0/mods-config/files/mab_users/authorize

RADIUS_CLIENT_SECRET="$(cat /run/secrets/radius-client/secret)"
RADIUS_LDAP_PASSWORD="$(cat /run/secrets/radius-ldap/password)"
export RADIUS_CLIENT_SECRET RADIUS_LDAP_PASSWORD

render_eap_vlan_map

exec /entrypoint.sh "$@"
