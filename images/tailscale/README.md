# Tailscale

The upstream Tailscale container with one storage optimization: `tailscale`
and `tailscaled` are a single, stripped binary selected by its invoked name.
The image keeps upstream `containerboot`, its Alpine base, packages, and
runtime interface.

The combined binary retains the full container feature set. It does not use
Tailscale's extra-small feature set or a binary packer.

Image versions track the Tailscale release used to build them.

## Build

```bash
docker build -t tailscale:test images/tailscale
docker run --rm tailscale:test tailscale version
docker run --rm tailscale:test tailscaled --version
```
