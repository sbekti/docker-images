#!/bin/sh
set -eu

copy_private_tree() {
    install -d -m 0750 "$2"
    cp -R "$1/." "$2/"
}

# RouterOS source mounts can be world-writable; use private runtime copies.
copy_private_tree /etc/raddb /run/radius-local-runtime/config
copy_private_tree /run/radius-local /run/radius-local-runtime/policy
copy_private_tree /run/secrets /run/radius-local-runtime/secrets
chown -R radius:radius /run/radius-local-runtime
chmod -R u=rwX,g=rX,o= /run/radius-local-runtime

exec "$@"
