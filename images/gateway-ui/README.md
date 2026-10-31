# gateway-ui

A small RouterOS Wi-Fi and Internet control panel. A static Go binary serves
HTML forms with no JavaScript or external assets.

Required environment variables are CONFIG_FILE (read-only JSON bindings),
ROUTER_ADDRESS (API endpoint with port), ROUTER_SERVER_NAME (TLS identity),
ROUTER_USER (API account), UI_HOST (permitted HTTP hosts, comma separated), and
TRUSTED_PREFIX (permitted client CIDRs, comma separated).

ROUTER_PASSWORD_FILE defaults to /run/secrets/router-password and ROUTER_CA_FILE
to /run/secrets/router-ca.pem. Mount these files read-only. Passwords are never
part of the image. LISTEN_ADDRESS defaults to :8080.

JSON contains station_profile, wan_list and radios. Each radio has interface,
uplink and an ordered aps array. Each AP has profile and, except for the first
binding, virtual_interface. Provision these objects and DHCP clients first.
Identity, SSIDs, bands, routes, addresses and DNS come from RouterOS.
The app toggles prepared clients; it does not choose routing or DNS policy.
Selections persist natively. Disconnected station radios remain reserved.

Every permitted client may operate the panel. Preserve client source addresses.
Host, source CIDR, Origin and CSRF guards protect forms. API permissions are
read,write,test,api, with the account restricted to its container address.
AP passwords and PPSK keys are never read.

Forms redirect to GET so reload never repeats an action. Only the latest scan
is cached in memory. No credentials are placed in redirect URLs or logs.

Validate with go test -race ./..., go vet ./..., and govulncheck ./....
PAGE_FILE optionally reloads a development template on every request.
Keep deployment configuration and credentials outside the public source tree.
