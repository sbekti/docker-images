# Samba RODC

A minimal Samba read-only domain controller for local authentication. Its
supported entrypoint has no writable-domain provisioning path.

Samba performs ordinary directory replication itself. Active Directory's
password replication policy decides which credentials are eligible for an
RODC, and permitted credentials are cached when authentication requires them.
The image does not implement its own replication scheduler, cache-selection
policy, or password-preload loop.

## Safety model

- Debian 13 backports must provide Samba 4.23.9 or newer, including the RODC
  fix required by this image.
- `run` opens the existing `sam.ldb` without creating it and requires Samba to
  identify it as read-only.
- Missing, corrupt, or writable-DC state is rejected before Samba starts.
- The runtime service set is fixed to `rpc, ldap, cldap, drepl, winbindd`.
- Dynamic RPC is fixed to TCP ports 50000–50019.
- DNS registration and DNS serving are disabled.
- Simple LDAP is permitted for authorization on an isolated container bridge;
  do not expose the LDAP listener to untrusted networks.
- No supported command provisions a writable domain.

The safety boundary is the supported entrypoint. Like any privileged,
root-running Samba image, an operator who replaces the entrypoint can invoke
the packaged Samba tools directly. Pin the image and entrypoint and restrict
who can change the container.

## Commands

| Command | Behavior |
|---|---|
| `join` | Interactively join empty persistent volumes as an RODC |
| `run` | Validate persisted RODC state and start Samba |
| `healthcheck` | Validate RODC state and ping local Samba and Winbind |

`run` and `healthcheck` use the persisted Samba configuration and require no
identity environment variables. Health deliberately remains good during a
central outage when local Samba and Winbind are usable.

At startup, `run` normalizes the ownership and permissions of Samba's mounted
socket directories and removes stale runtime files before starting Samba.

`join` requires:

| Variable | Meaning |
|---|---|
| `REALM` | AD DNS realm |
| `NETBIOS_NAME` | RODC NetBIOS name |
| `RODC_ADDRESS` | Address assigned to the RODC container |
| `RWDC_ADDRESS` | Writable DC address used for the one-time join |
| `JOIN_USER` | Plain AD join-account name |

Persistent state uses `/var/lib/samba` and `/etc/samba`. `/run/samba` contains
runtime sockets that a local authentication service can consume.

The container resolver must use domain DNS while central services are online.
Samba replication resolves its source through AD `_msdcs` GUID records.

## Credential policy

Configure password replication policy in Active Directory. The image neither
changes that policy nor chooses accounts itself. Removing an account from a
network-authorization group prevents future access after ordinary directory
replication, but it does not erase a previously revealed password from the
RODC database. Authorization must therefore check current group membership.
Discard and rejoin the RODC if local credential state must be purged.

## Join

Run `join` with a terminal. Samba prompts for the join account password without
echoing it. Passwords are not accepted through the supported command arguments.

```bash
docker run --rm -it --privileged \
  -e REALM \
  -e NETBIOS_NAME \
  -e RODC_ADDRESS \
  -e RWDC_ADDRESS \
  -e JOIN_USER \
  -v samba-var:/var/lib/samba \
  -v samba-etc:/etc/samba \
  samba-rodc:test join
```

Join is intentionally one-shot. It refuses existing `smb.conf` or `sam.ldb`
state. If enrollment leaves partial state, remove the stale RODC object
centrally, discard both local volumes, and join fresh. Do not repair, restore,
or promote the local database.

## Build

```bash
docker build -t samba-rodc:test images/samba-rodc
docker run --rm samba-rodc:test --help
```

Running without joined state must fail closed.
