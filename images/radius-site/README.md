# Site-local RADIUS

A small FreeRADIUS service for site-local Wi-Fi authentication.

- PEAP/MSCHAPv2 authenticates only against a local Samba RODC and fails closed.
- The first matching directory group in `RADIUS_EAP_VLAN_MAP` selects the VLAN.
- Authenticated users without a configured group are rejected.
- Non-EAP requests use a local, read-only MAC/VLAN snapshot.
- Unknown MACs and known MACs with a wrong credential receive the default VLAN.
- Accounting is not accepted or stored.

The image extends `ghcr.io/sbekti/freeradius:v3.2.8`. It has no database,
upstream RADIUS server, background synchronization process, or persistent
state of its own.

## Environment

| Variable | Purpose |
|---|---|
| `RADIUS_CLIENT_ADDRESS` | NAS address or subnet accepted by FreeRADIUS |
| `RADIUS_CLIENT_REQUIRE_MESSAGE_AUTHENTICATOR` | Require Attribute 80; default `yes` |
| `RADIUS_DEFAULT_VLAN` | VLAN returned for unknown or rejected non-EAP devices |
| `RADIUS_EAP_VLAN_MAP` | Ordered `group=vlan` pairs separated by commas |
| `RADIUS_LDAP_SERVER` | RODC LDAP address on the private container network |
| `RADIUS_LDAP_PORT` | RODC LDAP port; default `389` |
| `RADIUS_LDAP_IDENTITY` | Read-only LDAP service-account identity |
| `RADIUS_LDAP_BASE_DN` | LDAP search base |
| `RADIUS_WINBIND_DOMAIN` | Short domain used when an identity omits one |
| `RADIUS_LISTEN_ADDRESS` | Authentication listener; default `*` |
| `RADIUS_LISTEN_PORT` | Authentication port; default `1812` |
| `RADIUS_MAB_USERS_FILE` | Cached users file; default `/run/radius-site/mab-users` |

Map order is authorization priority. Group names may not contain commas or
equals signs. VLANs must be in the range 1–4094.

FreeRADIUS listens for authentication on UDP 1812. No request policy selects
an upstream server.

## Required mounts

| Path | Purpose |
|---|---|
| `/run/radius-site/mab-users` | Current snapshot in FreeRADIUS users-file format |
| `/run/samba` | RODC public Winbind socket directory |
| `/var/lib/samba/winbindd_privileged` | RODC privileged Winbind socket directory |
| `/etc/samba` | RODC Samba configuration |
| `/run/secrets/radius-client/secret` | NAS shared secret |
| `/run/secrets/radius-eap/server.key` | EAP server private key |
| `/run/secrets/radius-eap/server.pem` | EAP server certificate and chain |
| `/run/secrets/radius-ldap/password` | LDAP service-account password |

The snapshot must start with `# radius-site-mab-v1`. A marker-only file is a
valid empty snapshot. A missing, empty, or incorrectly marked snapshot stops
the container, so a first deployment cannot silently treat every device as
unknown. Runtime fetch failures should leave the last-known-good file in
place. The snapshot is imported when the container starts; restart the
container after replacing it.

The password and EAP private-key files must be readable at startup. The server
certificate file must contain any intermediate chain required by clients.
The EAP key and certificate are imported into private runtime files at startup.
Build this image and the RODC image with the same `WINBIND_PRIVILEGED_GID`;
the default is `998`.

LDAP uses a simple bind only on the private container network. Do not publish
the RODC LDAP listener or attach untrusted containers to that network.

```bash
docker build -t radius-site:test images/radius-site
```

## MAB integration test

From the repository root, run:

```bash
tests/radius-site-mab.sh
```

The test builds the image, starts it with temporary generic fixtures, sends
loopback RADIUS requests with `radclient`, verifies Guest and mapped VLANs, and
removes the container and fixtures when it exits.
