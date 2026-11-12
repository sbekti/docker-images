#!/bin/sh
# Test only the runtime contract; Homeinfra tests its rendered authentication policy.
set -eu
image=${1:-radius-local:test}
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
mkdir -p "$work/config" "$work/policy" "$work/secrets"
cat > "$work/config/radiusd.conf" <<'EOF'
prefix = /opt
libdir = /opt/lib
log {
    destination = stdout
}
security {
    user = radius
    group = radius
}
modules {
    $INCLUDE /opt/etc/raddb/mods-available/always
}
server example {
    listen {
        type = auth
        ipaddr = 127.0.0.1
        port = 1812
    }
    authorize {
        reject
    }
}
EOF
printf '%s' fixture > "$work/policy/policy"
chmod -R a+rwX "$work/config" "$work/policy" "$work/secrets"
run() {
    docker run --rm --network none \
        --tmpfs /tmp:rw,size=16m --tmpfs /run/radius-local-runtime:rw,size=4m \
        -v "$work/config:/etc/raddb:ro" \
        -v "$work/policy:/run/radius-local:ro" \
        -v "$work/secrets:/run/secrets:ro" "$image" "$@"
}
run radiusd -C -d /run/radius-local-runtime/config >/dev/null
run sh -ec '
    test "$(stat -c %a /run/radius-local-runtime/config/radiusd.conf)" = 640
    test "$(stat -c %a /run/radius-local-runtime/config)" = 750
    cmp /etc/raddb/radiusd.conf /run/radius-local-runtime/config/radiusd.conf
    cmp /run/radius-local/policy /run/radius-local-runtime/policy/policy
'
printf '%s\n' 'PASS native configuration and private RAM copies'
printf '%s\n' 'invalid configuration {' > "$work/config/radiusd.conf"
if run radiusd -C -d /run/radius-local-runtime/config >/dev/null 2>&1; then
    echo 'Invalid configuration was accepted' >&2
    exit 1
fi
printf '%s\n' 'PASS malformed configuration fails'
