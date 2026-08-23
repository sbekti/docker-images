# Samba RODC

A minimal Samba read-only domain controller for site-local Wi-Fi
authentication. Its supported entrypoint has no writable-domain provisioning
path.

This milestone provides safe, interactive RODC enrollment and restart. It is
not ready for deployment until M03 adds credential-cache reconciliation.

## Safety model

- Debian 13 backports must provide Samba 4.23.9 or newer, including the RODC
  fix required by this design.
- `run` opens the existing `sam.ldb` without creating it and requires Samba to
  identify it as read-only.
- Missing, corrupt, writable-DC, or configuration-mismatched state is rejected
  before Samba starts.
- The runtime service set is fixed to `rpc, ldap, cldap, drepl, winbindd, kcc`.
- Dynamic RPC is limited to TCP ports 50000–50019 by default.
- DNS registration and DNS serving are disabled. Site DNS stays authoritative
  outside this container.
- No supported command provisions a writable domain.

The safety boundary is the supported entrypoint. Like any privileged,
root-running Samba image, an operator who deliberately replaces the entrypoint
can invoke the packaged Samba tools directly. The deployment must therefore
pin the image and entrypoint and restrict who can change the container.

The container requires root and a privileged runtime because Samba AD DC
requires mount operations. The deployment owns privileges, mounts,
networking, resource limits, and digest pinning.

## Configuration

`join`, `run`, and `healthcheck` require these values:

| Variable | Meaning |
|---|---|
| `REALM` | Uppercase AD DNS realm |
| `DOMAIN` | Uppercase AD NetBIOS domain |
| `NETBIOS_NAME` | Uppercase RODC name, at most 15 characters |
| `DNS_HOSTNAME` | Exact lowercase name derived from the RODC name and realm |
| `RODC_ADDRESS` | Literal IPv4 address assigned to the RODC |

`RPC_PORT_START` and `RPC_PORT_END` optionally replace the default
`50000–50019` range. `join` also requires the direct writable-DC IPv4 address
in `RWDC_ADDRESS` and a plain AD account name in `JOIN_USER`.

## Commands

| Command | Behavior |
|---|---|
| `run` | Validate existing RODC state and start the fixed service set |
| `join` | Interactively join empty persistent volumes as an RODC |
| `sync-once` | Refuse with exit status 78 until M03 |
| `healthcheck` | Validate RODC state and ping the running Samba process |

Persistent state uses `/var/lib/samba` and `/etc/samba`. `/run/samba` contains
runtime sockets that a local FreeRADIUS container will consume.

## Join

Run `join` with a terminal. Samba prompts for the join account password without
echoing it. The entrypoint rejects password-bearing environment variables and
does not support passwords in arguments or files.

```bash
docker run --rm -it --privileged \
  -e REALM \
  -e DOMAIN \
  -e NETBIOS_NAME \
  -e DNS_HOSTNAME \
  -e RODC_ADDRESS \
  -e RWDC_ADDRESS \
  -e JOIN_USER \
  -v samba-var:/var/lib/samba \
  -v samba-etc:/etc/samba \
  samba-rodc:m02 join
```

Join is intentionally one-shot. It refuses any existing `smb.conf` or
`sam.ldb`. If a failed enrollment leaves partial state, remove the stale RODC
computer object centrally, discard both local volumes, and join fresh. Do not
try to repair or promote the local database.

## Build

```bash
docker build -t samba-rodc:m02 images/samba-rodc
docker run --rm samba-rodc:m02 --help
```

Running without joined state or required configuration must fail closed.
