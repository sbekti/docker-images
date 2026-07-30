# RADIUS proxy

A small, stateless FreeRADIUS authentication proxy.

Policy:

- Requests containing `EAP-Message` go only to the primary.
- Other authentication requests use primary, then secondary.
- An explicit non-EAP reject becomes an accept on the default VLAN.
- A newly timed-out request receives no response so the NAS can retry.
- Duplicate cleanup is disabled so that exact NAS retransmission is processed
  as a new request and can use the next live authority.
- When both non-EAP authorities are marked dead, a local fallback returns the
  default VLAN.
- Primary Status-Server probes restore the preferred path after an outage.

The image extends `ghcr.io/sbekti/freeradius:v3.2.8`. It adds no custom
entrypoint, database, or persistent state.

The published `radius-proxy:v1.0.0` image reports FreeRADIUS 3.2.5 at runtime,
matching the current IAD2 service. Rebuilding both services on 3.2.8 is
deferred; the image tag alone must not be treated as the daemon version.

## Environment

| Variable | Purpose |
|---|---|
| `RADIUS_CLIENT_ADDRESS` | NAS address or subnet accepted by the proxy |
| `RADIUS_CLIENT_SECRET` | NAS-to-proxy shared secret |
| `RADIUS_CLIENT_REQUIRE_MESSAGE_AUTHENTICATOR` | Require Attribute 80 from downstream clients; default `yes` |
| `RADIUS_LISTEN_ADDRESS` | Authentication listener address; default `*` |
| `RADIUS_LISTEN_PORT` | Authentication listener port; default `1812` |
| `RADIUS_CLEANUP_DELAY` | Seconds to cache a completed request; default `0` so exact retransmissions can fail over |
| `RADIUS_PRIMARY_ADDRESS` | Preferred RADIUS server |
| `RADIUS_PRIMARY_PORT` | Primary authentication port; default `1812` |
| `RADIUS_PRIMARY_SECRET` | Proxy-to-primary shared secret |
| `RADIUS_SECONDARY_ADDRESS` | Secondary RADIUS server |
| `RADIUS_SECONDARY_PORT` | Secondary authentication port; default `1812` |
| `RADIUS_SECONDARY_SECRET` | Proxy-to-secondary shared secret |
| `RADIUS_DEFAULT_VLAN` | VLAN returned for non-EAP rejects |
| `RADIUS_RESPONSE_WINDOW` | Shared response window; default `2` |
| `RADIUS_RESPONSE_TIMEOUTS` | Shared response timeout count; default `1` |
| `RADIUS_ZOMBIE_PERIOD` | Shared zombie period; default `20` |
| `RADIUS_PRIMARY_STATUS_CHECK` | Primary health-check mode; default `status-server` |
| `RADIUS_PRIMARY_CHECK_INTERVAL` | Primary check interval; default `10` |
| `RADIUS_PRIMARY_CHECK_TIMEOUT` | Primary check timeout; default `3` |
| `RADIUS_PRIMARY_NUM_ANSWERS_TO_ALIVE` | Successful checks required for recovery; default `3` |
| `RADIUS_SECONDARY_STATUS_CHECK` | Secondary health-check mode; default `none` |
| `RADIUS_SECONDARY_REVIVE_INTERVAL` | Secondary revive interval; default `60` |

Client and upstream addresses, shared secrets, and the default VLAN are
required. Listener and health behavior have image defaults and can be
overridden independently at runtime.

Build-time configuration validation uses documentation-only addresses and
secrets:

```bash
docker build -t radius-proxy:test images/radius-proxy
```
