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
    : > /run/radius-local-eap-vlan

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
            >> /run/radius-local-eap-vlan
    done

    chmod 0644 /run/radius-local-eap-vlan
)

render_operator_reply_attributes() (
    IFS=,
    output=/run/radius-local-operator-reply
    previous=
    printf '%s\n' DEFAULT > "${output}"

    for mapping in ${RADIUS_OPERATOR_REPLY_ATTRIBUTES}; do
        attribute="${mapping%%=*}"
        value="${mapping#*=}"

        [ -n "${value}" ] && [ "${value}" != "${mapping}" ] \
            || fail "RADIUS_OPERATOR_REPLY_ATTRIBUTES must contain Attribute=value entries."
        case "${attribute}" in
            [A-Za-z]*) ;;
            *) fail "Invalid attribute in RADIUS_OPERATOR_REPLY_ATTRIBUTES." ;;
        esac
        case "${attribute}" in
            *[!A-Za-z0-9-]*) fail "Invalid attribute in RADIUS_OPERATOR_REPLY_ATTRIBUTES." ;;
        esac

        escaped_value="$(printf '%s' "${value}" | sed 's/\\/\\\\/g; s/"/\\"/g')"
        if [ -n "${previous}" ]; then
            printf '%s,\n' "${previous}" >> "${output}"
        fi
        previous="    ${attribute} += \"${escaped_value}\""
    done

    [ -n "${previous}" ] || fail "RADIUS_OPERATOR_REPLY_ATTRIBUTES is empty."
    printf '%s\n' "${previous}" >> "${output}"
    chmod 0644 "${output}"
)

: "${RADIUS_CLIENT_ADDRESS:?RADIUS_CLIENT_ADDRESS is required.}"
: "${RADIUS_OPERATOR_CLIENT_ADDRESS:?RADIUS_OPERATOR_CLIENT_ADDRESS is required.}"
: "${RADIUS_OPERATOR_GROUP:?RADIUS_OPERATOR_GROUP is required.}"
: "${RADIUS_OPERATOR_REPLY_ATTRIBUTES:?RADIUS_OPERATOR_REPLY_ATTRIBUTES is required.}"
: "${RADIUS_DEFAULT_VLAN:?RADIUS_DEFAULT_VLAN is required.}"
: "${RADIUS_EAP_VLAN_MAP:?RADIUS_EAP_VLAN_MAP is required.}"
: "${RADIUS_LDAP_BASE_DN:?RADIUS_LDAP_BASE_DN is required.}"
: "${RADIUS_LDAP_IDENTITY:?RADIUS_LDAP_IDENTITY is required.}"
: "${RADIUS_LDAP_SERVER:?RADIUS_LDAP_SERVER is required.}"
: "${RADIUS_NTLM_AUTH_BRIDGE_URL:?RADIUS_NTLM_AUTH_BRIDGE_URL is required.}"
: "${RADIUS_WINBIND_DOMAIN:?RADIUS_WINBIND_DOMAIN is required.}"

valid_vlan "${RADIUS_DEFAULT_VLAN}" || fail "RADIUS_DEFAULT_VLAN must be 1-4094."

require_file /run/secrets/radius-client/secret
require_file /run/secrets/radius-operator/secret
require_file /run/secrets/radius-eap/server.pem
require_file /run/secrets/radius-ldap/password
require_file "${RADIUS_MAB_USERS_FILE}"
IFS= read -r marker < "${RADIUS_MAB_USERS_FILE}" || true
[ "${marker:-}" = "# radius-mab-v1" ] \
    || fail "Invalid MAB snapshot marker in ${RADIUS_MAB_USERS_FILE}."
install -d -m 0750 -o radius -g radius /run/radius-local-eap
install -m 0640 -o radius -g radius /run/secrets/radius-eap/server.pem \
    /run/radius-local-eap/server.pem
install -m 0644 "${RADIUS_MAB_USERS_FILE}" \
    /opt/etc/raddb/mods-config/files/mab_users/authorize

RADIUS_CLIENT_SECRET="$(cat /run/secrets/radius-client/secret)"
RADIUS_OPERATOR_CLIENT_SECRET="$(cat /run/secrets/radius-operator/secret)"
RADIUS_LDAP_PASSWORD="$(cat /run/secrets/radius-ldap/password)"
export RADIUS_CLIENT_SECRET RADIUS_LDAP_PASSWORD RADIUS_OPERATOR_CLIENT_SECRET

render_eap_vlan_map
render_operator_reply_attributes

if [ "${1:-}" = radiusd ] || [ "${1:-}" = freeradius ]; then
    shift
    set -- radiusd -f "$@"
fi

exec "$@"
