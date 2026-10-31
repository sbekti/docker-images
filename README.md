# docker-images

Docker image builds, published to `ghcr.io/sbekti/<image>`.

## Images

| Image | Description |
|-------|-------------|
| [asterisk](images/asterisk/) | Asterisk PBX |
| [hermes-tools](images/hermes-tools/) | Hermes MCP tools and generic seed skills |
| [gateway-ui](images/gateway-ui/) | Local RouterOS control panel for Wi-Fi and Internet settings |
| [ntlm-auth-bridge](images/ntlm-auth-bridge/) | Private HTTP adapter for Samba MS-CHAPv2 authentication |
| [radius-local](images/radius-local/) | Local Wi-Fi and network-device authentication |
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
Documentation-only pushes skip the workflow. Release tags and manual builds
remain available regardless of changed paths.

Pull requests that change an image build that image for both `linux/amd64` and
`linux/arm64` without publishing it. A change to the shared build workflow
builds every image. The final `Image build gate` job summarizes the required
matrix result.

New commits to a pull request cancel its older builds. Publishing and manual
runs remain independent.

Renovate checks Docker base images, GitHub Actions, Python requirements, and Go modules.
Routine updates run on Fridays in America/New_York. Grouped patches may
automerge after CI passes; grouped minors and separate majors need review.
Recognized GitHub security alerts bypass the weekly schedule.

FreeRADIUS source versions and checksums, Samba package minimums, and runtime
OS packages need release or distribution checks rather than version-only PRs.
Published version tags must exist before homeinfra updates its consumers.
