# NTLM authentication bridge

A small HTTP adapter for MS-CHAPv2 authentication through Samba's
`ntlm_auth`. It is intended for a private container network alongside a Samba
domain controller.

The bridge accepts `POST /auth` requests:

```json
{
  "username": "alice",
  "domain": "EXAMPLE",
  "challenge": "0123456789abcdef",
  "nt_response": "0123456789abcdef0123456789abcdef0123456789abcdef",
  "request_nt_key": true
}
```

Successful authentication returns:

```json
{"authenticated":true,"nt_key":"00000000000000000000000000000000"}
```

Rejected credentials return HTTP 200 with `{"authenticated":false}`.
Malformed requests and service failures use non-200 responses. The bridge
does not log request contents or return output from `ntlm_auth`.

## Options

| Option | Default | Purpose |
|---|---|---|
| `--port` | `9555` | HTTP listen port |
| `--ntlm-auth-path` | `ntlm_auth` | Path to the Samba helper |
| `--timeout` | `2s` | Maximum helper runtime |

The container must have access to the privileged Winbind socket. Do not
publish the HTTP listener or attach untrusted containers to its network.
