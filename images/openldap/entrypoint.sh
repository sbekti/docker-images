#!/bin/bash
set -exo pipefail

LDAP_MAX_OPEN_FILES="${LDAP_MAX_OPEN_FILES:-1024}"
LDAP_ADMIN_PASSWORD="${LDAP_ADMIN_PASSWORD:-password}"
LDAP_DOMAIN="${LDAP_DOMAIN:-example.com}"
LDAP_ORGANIZATION="${LDAP_ORGANIZATION:-Example Org}"
LDAP_FORCE_RECONFIGURE="${LDAP_FORCE_RECONFIGURE:-false}"

# When not limiting the open file descritors limit, the memory consumption of
# slapd is absurdly high. See https://github.com/docker/docker/issues/8231
ulimit -n ${LDAP_MAX_OPEN_FILES}

first_run=true

if [[ -f "/var/lib/ldap/data.mdb" || -f "/etc/ldap/slapd.d/cn=config.ldif" ]]; then
    first_run=false
fi

if [[ "$LDAP_FORCE_RECONFIGURE" == "true" ]]; then
    first_run=true
fi

echo "First run: $first_run"

if [[ "$first_run" == "true" ]]; then
    cat <<EOF | debconf-set-selections
slapd slapd/internal/generated_adminpw password ${LDAP_ADMIN_PASSWORD}
slapd slapd/internal/adminpw password ${LDAP_ADMIN_PASSWORD}
slapd slapd/password2 password ${LDAP_ADMIN_PASSWORD}
slapd slapd/password1 password ${LDAP_ADMIN_PASSWORD}
slapd slapd/dump_database_destdir string /var/backups/slapd-VERSION
slapd slapd/domain string ${LDAP_DOMAIN}
slapd shared/organization string ${LDAP_ORGANIZATION}
slapd slapd/purge_database boolean true
slapd slapd/move_old_database boolean true
slapd slapd/allow_ldap_v2 boolean false
slapd slapd/no_configuration boolean false
slapd slapd/dump_database select when needed
EOF

    dpkg-reconfigure -f noninteractive slapd

    if [[ -d "/etc/ldap/prepopulate/schemas" ]]; then
        for file in `ls /etc/ldap/prepopulate/schemas/*.ldif`; do
            slapadd -n0 -F /etc/ldap/slapd.d -l "$file"
        done
    fi

    if [[ -d "/etc/ldap/prepopulate/modules" ]]; then
        for file in `ls /etc/ldap/prepopulate/modules/*.ldif`; do
            slapmodify -n0 -F /etc/ldap/slapd.d -l "$file"
        done
    fi

    if [[ -d "/etc/ldap/prepopulate/data" ]]; then
        for file in `ls /etc/ldap/prepopulate/data/*.ldif`; do
            slapadd -F /etc/ldap/slapd.d -l "$file"
        done
    fi
fi

exec "$@"
