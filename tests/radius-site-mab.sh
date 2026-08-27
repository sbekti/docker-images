#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IMAGE="${RADIUS_SITE_TEST_IMAGE:-radius-site:mab-test}"
CONTAINER="radius-site-mab-test-${RANDOM}"
FIXTURE="$(mktemp -d)"
SECRET="test-radius-secret"
MAC="020000000001"

cleanup() {
    docker rm --force "${CONTAINER}" >/dev/null 2>&1 || true
    rm -rf "${FIXTURE}"
}
trap cleanup EXIT

for command in docker openssl; do
    command -v "${command}" >/dev/null || {
        echo "Missing required command: ${command}" >&2
        exit 1
    }
done

mkdir -p "${FIXTURE}/mab" "${FIXTURE}/radius-client" "${FIXTURE}/radius-eap" "${FIXTURE}/radius-ldap"
chmod 0755 "${FIXTURE}" "${FIXTURE}/mab" "${FIXTURE}/radius-client" "${FIXTURE}/radius-eap" "${FIXTURE}/radius-ldap"
printf '%s\n' "${SECRET}" > "${FIXTURE}/radius-client/secret"
printf '%s\n' 'test-ldap-password' > "${FIXTURE}/radius-ldap/password"
openssl req -x509 -newkey rsa:2048 -nodes -days 1 -subj '/CN=radius.example.test' \
    -keyout "${FIXTURE}/radius-eap/server.key" \
    -out "${FIXTURE}/radius-eap/server.pem" >/dev/null 2>&1
chmod 0666 "${FIXTURE}/radius-eap/server.key" "${FIXTURE}/radius-eap/server.pem"

write_snapshot() {
    local vlan="$1"
    printf '%s\n' \
        '# radius-site-mab-v1' \
        "${MAC} Cleartext-Password := \"${MAC}\"" \
        '    Tunnel-Type := VLAN,' \
        '    Tunnel-Medium-Type := IEEE-802,' \
        "    Tunnel-Private-Group-Id := \"${vlan}\"" \
        > "${FIXTURE}/mab/mab-users"
    chmod 0666 "${FIXTURE}/mab/mab-users"
}

start_radius() {
    docker rm --force "${CONTAINER}" >/dev/null 2>&1 || true
    docker run --detach --name "${CONTAINER}" \
        --env RADIUS_CLIENT_ADDRESS=127.0.0.1 \
        --env RADIUS_DEFAULT_VLAN=10 \
        --env RADIUS_EAP_VLAN_MAP=wifi-primary=1 \
        --env RADIUS_LDAP_BASE_DN=dc=example,dc=test \
        --env RADIUS_LDAP_IDENTITY=radius-reader@example.test \
        --env RADIUS_LDAP_SERVER=127.0.0.1 \
        --env RADIUS_WINBIND_DOMAIN=EXAMPLE \
        --mount "type=bind,src=${FIXTURE}/mab,dst=/run/radius-site,readonly" \
        --mount "type=bind,src=${FIXTURE}/radius-client,dst=/run/secrets/radius-client,readonly" \
        --mount "type=bind,src=${FIXTURE}/radius-eap,dst=/run/secrets/radius-eap,readonly" \
        --mount "type=bind,src=${FIXTURE}/radius-ldap,dst=/run/secrets/radius-ldap,readonly" \
        "${IMAGE}" >/dev/null

    for _ in {1..40}; do
        if docker exec "${CONTAINER}" grep -q 'Ready to process requests' /var/log/freeradius/radius.log 2>/dev/null; then
            return
        fi
        if ! docker inspect --format '{{.State.Running}}' "${CONTAINER}" | grep -q true; then
            docker cp "${CONTAINER}:/var/log/freeradius/radius.log" "${FIXTURE}/radius.log" >/dev/null 2>&1 || true
            sed -n '1,200p' "${FIXTURE}/radius.log" >&2 2>/dev/null || true
            return 1
        fi
        sleep 0.25
    done
    docker exec "${CONTAINER}" sed -n '1,200p' /var/log/freeradius/radius.log >&2 || true
    return 1
}

assert_mab() {
    local username="$1"
    local password="$2"
    local vlan="$3"
    local response
    response="$(printf '%s\n' \
        "User-Name = \"${username}\"" \
        "User-Password = \"${password}\"" \
        'Message-Authenticator = 0x00' \
        | docker exec --interactive "${CONTAINER}" radclient -x 127.0.0.1 auth "${SECRET}")"
    if ! grep -q 'Access-Accept' <<< "${response}" \
        || ! grep -q "Tunnel-Private-Group-Id:0 = \"${vlan}\"" <<< "${response}"; then
        echo "${response}" >&2
        return 1
    fi
}

docker build --tag "${IMAGE}" "${ROOT}/images/radius-site" >/dev/null

write_snapshot 20
start_radius
assert_mab "${MAC}" "${MAC}" 20
assert_mab "${MAC}" 'wrong-password' 10
assert_mab '020000000099' '020000000099' 10

write_snapshot 1
start_radius
assert_mab "${MAC}" "${MAC}" 1

echo 'radius-site MAB integration test passed'
