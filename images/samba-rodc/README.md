# Samba RODC

A minimal Samba read-only domain controller for site-local Wi-Fi
authentication. Its supported entrypoint intentionally has no writable-domain
provisioning path.

This milestone provides the image and safety boundary only. It is not ready
for deployment: the interactive join arrives in M02 and credential-cache
reconciliation in M03.

## Safety model

- Debian 13 backports must provide Samba 4.23.9 or newer, including the RODC
  fix required by this design.
- `run` opens an existing `sam.ldb` with Samba's `FLG_DONT_CREATE_DB` flag and
  requires `SamDB.am_rodc()` to return true.
- Missing, corrupt, or writable-DC state is rejected before Samba starts.
- The runtime service set is fixed to `rpc, ldap, cldap, drepl, winbindd, kcc`.
- No command provisions a domain.

The safety boundary is the supported entrypoint. Like any privileged,
root-running Samba image, an operator who deliberately replaces the entrypoint
can invoke the packaged Samba tools directly. The deployment must therefore
pin the image and entrypoint and restrict who can change the container.

The container requires root and a privileged RouterOS deployment because the
Samba AD DC runtime requires mount operations. Deployment privileges, mounts,
networking, resource limits, and digest pinning are owned by `homeinfra`.

## Commands

| Command | M01 behavior |
|---|---|
| `run` | Validate existing RODC state and start the fixed service set |
| `join` | Refuse with exit status 78 until M02 |
| `sync-once` | Refuse with exit status 78 until M03 |
| `healthcheck` | Validate RODC state and ping the running Samba process |

Persistent state uses `/var/lib/samba` and `/etc/samba`. `/run/samba` contains
runtime sockets that a local FreeRADIUS container will consume.

## Build

```bash
docker build -t samba-rodc:m01 images/samba-rodc
docker run --rm samba-rodc:m01 --help
```

Running without joined state must fail:

```bash
docker run --rm samba-rodc:m01
```
