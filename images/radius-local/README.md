# Local RADIUS

A small FreeRADIUS service for local network authentication.

- PEAP/MSCHAPv2 authenticates only against a local Samba RODC and fails closed.
- The first matching directory group in `RADIUS_EAP_VLAN_MAP` selects the VLAN.
- Authenticated users without a configured group are rejected.
- Non-EAP requests use a local, read-only MAC/VLAN snapshot.
- Unknown MACs and known MACs with a wrong credential receive the default VLAN.
- Accounting is not accepted or stored.
- A separate PAP-only listener authenticates human operators through
  LDAP and requires membership in one configured directory group.
- Successful operator-access requests receive one configurable set of
  RADIUS reply attributes.

The image builds FreeRADIUS 3.2.10 from the pinned upstream source. It has no
database, upstream RADIUS server, background synchronization process, or
persistent state of its own.

## Environment

| Variable | Purpose |
|---|---|
| `RADIUS_CLIENT_ADDRESS` | NAS address or subnet accepted by FreeRADIUS |
| `RADIUS_CLIENT_REQUIRE_MESSAGE_AUTHENTICATOR` | Require Attribute 80; default `yes` |
| `RADIUS_OPERATOR_CLIENT_ADDRESS` | Network-device address/subnet accepted by the operator-access listener |
| `RADIUS_OPERATOR_CLIENT_REQUIRE_MESSAGE_AUTHENTICATOR` | Require Attribute 80; default `yes` |
| `RADIUS_OPERATOR_GROUP` | Directory group authorized for operator access |
| `RADIUS_OPERATOR_LISTEN_ADDRESS` | Operator-access listener; default `*` |
| `RADIUS_OPERATOR_LISTEN_PORT` | Operator-access port; default `18120` |
| `RADIUS_OPERATOR_REPLY_ATTRIBUTES` | Comma-separated `Attribute=value` entries returned after successful authentication |
| `RADIUS_DEFAULT_VLAN` | VLAN returned for unknown or rejected non-EAP devices |
| `RADIUS_EAP_VLAN_MAP` | Ordered `group=vlan` pairs separated by commas |
| `RADIUS_LDAP_SERVER` | RODC LDAP address on the private container network |
| `RADIUS_LDAP_PORT` | RODC LDAP port; default `389` |
| `RADIUS_LDAP_IDENTITY` | Read-only LDAP service-account identity |
| `RADIUS_LDAP_BASE_DN` | LDAP search base |
| `RADIUS_NTLM_AUTH_BRIDGE_URL` | Local NTLM authentication bridge base URL |
| `RADIUS_WINBIND_DOMAIN` | Short domain used when an identity omits one |
| `RADIUS_LISTEN_ADDRESS` | Authentication listener; default `*` |
| `RADIUS_LISTEN_PORT` | Authentication port; default `1812` |
| `RADIUS_MAB_USERS_FILE` | Cached users file; default `/run/radius-local/mab-users` |

Map order is authorization priority. Group names may not contain commas or
equals signs. VLANs must be in the range 1–4094.

Operator reply values may contain equals signs; commas separate
entries. Attributes are added with `+=`, so an attribute may be repeated when
a device requires multiple values. Attribute names and types must exist in
the FreeRADIUS dictionaries included in the image.

FreeRADIUS listens for Wi-Fi authentication on UDP 1812 and operator
authentication on UDP 18120. The operator-access listener accepts only PAP
and uses its own client secret. No request policy selects an upstream server.

## Required mounts

| Path | Purpose |
|---|---|
| `/run/radius-local/mab-users` | Current snapshot in FreeRADIUS users-file format |
| `/run/secrets/radius-operator/secret` | Operator-access RADIUS shared secret |
| `/run/secrets/radius-client/secret` | NAS shared secret |
| `/run/secrets/radius-eap/server.pem` | EAP server certificate, chain, and private key |
| `/run/secrets/radius-ldap/password` | LDAP service-account password |

The snapshot must start with `# radius-mab-v1`. A marker-only file is a
valid empty snapshot. A missing, empty, or incorrectly marked snapshot stops
the container, so a first deployment cannot silently treat every device as
unknown. Runtime fetch failures should leave the last-known-good file in
place. The snapshot is imported when the container starts; restart the
container after replacing it.

The password and EAP private-key files must be readable at startup. The server
certificate file must contain any intermediate chain required by clients.
The EAP key and certificate are imported into private runtime files at startup.

LDAP uses a simple bind only on the private container network. Do not publish
the RODC LDAP listener or attach untrusted containers to that network.

```bash
docker build -t radius-local:test images/radius-local
```
