# docker-images

Docker image builds, published to `ghcr.io/sbekti/<image>`.

## Images

| Image | Description |
|-------|-------------|
| [asterisk](images/asterisk/) | Asterisk PBX |
| [hermes-tools](images/hermes-tools/) | Hermes MCP tools and generic seed skills |
| [ntlm-auth-bridge](images/ntlm-auth-bridge/) | Private HTTP adapter for Samba MS-CHAPv2 authentication |
| [radius-local](images/radius-local/) | Local EAP and cached MAC authentication |
| [samba-ad](images/samba-ad/) | Samba Active Directory DC |
| [samba-rodc](images/samba-rodc/) | Minimal read-only AD DC for local Wi-Fi authentication |
| [tailscale](images/tailscale/) | Upstream-compatible Tailscale with a combined CLI and daemon binary |

## Usage

```bash
docker pull ghcr.io/sbekti/radius-local:v2.0.0
```

## Releasing a New Version

Tag with the image name prefix:

```bash
git tag asterisk/v1.0.0
git push origin asterisk/v1.0.0
```

This builds and pushes `ghcr.io/sbekti/asterisk:v1.0.0` and `:latest`.

Pushing to `main` automatically builds any images whose files changed, tagged as `:main`.

Pull requests that change an image build that image for both `linux/amd64` and
`linux/arm64` without publishing it. A change to the shared build workflow
builds every image. The final `Image build gate` job summarizes the required
matrix result.

Renovate checks Docker base images and GitHub Actions. Grouped patch updates
run on Fridays and may automerge after CI passes, grouped minor updates run on
Saturdays for review, and major updates receive separate reviewable pull
requests on Sundays.
