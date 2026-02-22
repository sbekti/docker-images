# docker-images

Docker image builds, published to `ghcr.io/sbekti/<image>`.

## Images

| Image | Description |
|-------|-------------|
| [asterisk](images/asterisk/) | Asterisk PBX |
| [aws-cli-tgz](images/aws-cli-tgz/) | AWS CLI v2 |
| [dnsmasq](images/dnsmasq/) | dnsmasq DNS/DHCP server |
| [freeradius](images/freeradius/) | FreeRADIUS server |
| [novnc](images/novnc/) | noVNC web client |
| [openldap](images/openldap/) | OpenLDAP server |
| [samba-ad](images/samba-ad/) | Samba Active Directory DC |

## Usage

```bash
docker pull ghcr.io/sbekti/asterisk:latest
```

## Releasing a New Version

Tag with the image name prefix:

```bash
git tag asterisk/v1.0.0
git push origin asterisk/v1.0.0
```

This builds and pushes `ghcr.io/sbekti/asterisk:v1.0.0` and `:latest`.

Pushing to `main` automatically builds any images whose files changed, tagged as `:main`.
