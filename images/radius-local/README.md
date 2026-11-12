# FreeRADIUS runtime

A small FreeRADIUS image for RouterOS and Docker. The image supplies the server,
its selected modules, and the NTLM bridge client. Deployments supply the complete
native FreeRADIUS configuration. No site settings or credentials belong in the image.

## Inputs

Mount these directories read-only:

| Container path | Contents |
| --- | --- |
| `/etc/raddb` | Complete native configuration, including `radiusd.conf` |
| `/run/radius-local` | Policy data referenced by the configuration |
| `/run/secrets` | Certificates and private keys referenced by the configuration |

Provide bounded RAM filesystems at `/run/radius-local-runtime` (4 MiB) and
`/tmp` (16 MiB). At startup the image copies the three inputs into `config`,
`policy`, and `secrets` under the runtime directory, with private permissions.
This accommodates RouterOS source mounts that expose world-writable files.
Source files remain untouched. Refer to the runtime paths in configuration.

The default command runs FreeRADIUS in the foreground against the runtime
configuration. Pass a complete command to override it, for example:

```bash
docker run --rm \
  --mount type=bind,src=/srv/radius/config,dst=/etc/raddb,readonly \
  --mount type=bind,src=/srv/radius/policy,dst=/run/radius-local,readonly \
  --mount type=bind,src=/srv/radius/secrets,dst=/run/secrets,readonly \
  --tmpfs /run/radius-local-runtime:size=4m \
  --tmpfs /tmp:size=16m \
  radius-local:test radiusd -C -d /run/radius-local-runtime/config
```

The image has no environment-variable configuration interface, policy generator,
or synchronization process. A deployment must install a valid configuration
before starting it. Keep secrets out of normal logs; debug output may expose them.

The RouterOS-verified Alpine base is retained. Newer bases require a device smoke
test because Alpine 3.24.3 failed before application startup on RouterOS 7.24.5.

```bash
docker build -t radius-local:test images/radius-local
sh images/radius-local/test.sh
```

Authentication policy and its tests belong to the deployment supplying the configuration.
