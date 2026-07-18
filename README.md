# docker-images

Docker image builds, published to `ghcr.io/sbekti/<image>`.

## Images

| Image | Description |
|-------|-------------|
| [asterisk](images/asterisk/) | Asterisk PBX |
| [aws-cli-tgz](images/aws-cli-tgz/) | AWS CLI v2 |
| [freeradius](images/freeradius/) | FreeRADIUS server |
| [hermes-tools](images/hermes-tools/) | Hermes MCP tools and generic seed skills |
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

Pull requests that change an image build that image for both `linux/amd64` and
`linux/arm64` without publishing it. A change to the shared build workflow
builds every image. The final `Image build gate` job summarizes the required
matrix result.

Renovate checks Docker base images and GitHub Actions. Grouped patch updates
run on Fridays and may automerge after CI passes, grouped minor updates run on
Saturdays for review, and major updates receive separate reviewable pull
requests on Sundays. See [BEST_PRACTICES.md](BEST_PRACTICES.md) for the current
audit and deferred work.
