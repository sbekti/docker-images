#!/bin/sh
# Exercise the published reply policy with synthetic MAC identities.
set -eu
image=${1:-radius-local:test}
work=$(mktemp -d)
container="radius-local-test-$$"
trap 'docker rm -f "$container" >/dev/null 2>&1 || true; rm -rf "$work"' EXIT
openssl req -x509 -newkey rsa:2048 -nodes -days 1 -subj /CN=radius.example.test \
    -keyout "$work/key.pem" -out "$work/server.pem" >/dev/null 2>&1
cat "$work/key.pem" >> "$work/server.pem"
chmod 644 "$work/server.pem"
docker run -d --name "$container" --network none \
    -v "$work/server.pem:/run/secrets/radius-eap/server.pem:ro" \
    -e RADIUS_CLIENT_ADDRESS=127.0.0.1 \
    -e RADIUS_OPERATOR_CLIENT_ADDRESS=192.0.2.1 \
    -e RADIUS_OPERATOR_GROUP=operators \
    -e RADIUS_OPERATOR_REPLY_ATTRIBUTES=Filter-Id=operator \
    -e RADIUS_DEFAULT_VLAN=10 \
    -e RADIUS_EAP_VLAN_MAP=trusted=100,iot=20 \
    -e RADIUS_LDAP_BASE_DN=dc=example,dc=test \
    -e RADIUS_LDAP_IDENTITY=reader@example.test \
    -e RADIUS_LDAP_SERVER=192.0.2.10 \
    -e RADIUS_NTLM_AUTH_BRIDGE_URL=http://192.0.2.11:9555 \
    -e RADIUS_WINBIND_DOMAIN=EXAMPLE \
    --entrypoint /bin/sh "$image" -ec '
        mkdir -p /run/radius-local /run/secrets/radius-client /run/secrets/radius-operator /run/secrets/radius-ldap
        for name in radius-client radius-operator radius-ldap; do
            printf "%s\n" test-secret > /run/secrets/$name/secret
        done
        mv /run/secrets/radius-ldap/secret /run/secrets/radius-ldap/password
        printf "%s\n" "# radius-mab-v1" > /run/radius-local/mab-users
        for entry in 020000000001:100 020000000002:20; do
            mac=${entry%:*}; vlan=${entry#*:}
            printf "%s Cleartext-Password := \"%s\"\n    Tunnel-Type := VLAN,\n    Tunnel-Medium-Type := IEEE-802,\n    Tunnel-Private-Group-Id := \"%s\"\n\n" "$mac" "$mac" "$vlan" >> /run/radius-local/mab-users
        done
        exec /radius-local-entrypoint.sh radiusd
    ' >/dev/null
for attempt in 1 2 3 4 5; do
    if docker logs "$container" 2>&1 | grep -q 'Ready to process requests'; then break; fi
    sleep 1
done
check() {
    reply=$(printf 'User-Name = "%s"\nUser-Password = "%s"\n' "$1" "$2" |
        docker exec -i "$container" radclient -x -r 1 -t 3 -S /run/secrets/radius-client/secret 127.0.0.1:1812 auth)
    printf '%s\n' "$reply" | grep -q 'Received Access-Accept'
    printf '%s\n' "$reply" | grep -Eq "Tunnel-Private-Group-Id(:0)? = \"$3\""
    printf '%s\n' "$reply" | grep -Eq "Mikrotik-Wireless-VLANID = $3$"
    printf '%s\n' "$reply" | grep -Eq 'Mikrotik-Wireless-VLANID(-Type|type) = (0|802.1q)$'
    printf 'PASS %s\n' "$4"
}
check 020000000001 020000000001 100 trusted
check 020000000002 020000000002 20 iot
check 020000000003 020000000003 10 unknown
check 020000000001 wrong-password 10 wrong-credential
