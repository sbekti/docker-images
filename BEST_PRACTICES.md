# Container image best-practices audit

This is the repository checklist and findings inventory as of 2026-07-18.
It records work; it does not silently change image behavior. Image-specific
fixes should be reviewed and released independently.

## Repository checklist

### Dependency and build automation

- [x] Renovate is configured to scan only Dockerfile base images and GitHub
  Actions.
- [x] Renovate is configured to create a Dependency Dashboard.
- [x] Grouped patch updates run on Fridays and may automerge only after
  available CI checks pass; grouped minor updates run on Saturdays for review;
  major updates receive separate reviewable PRs on Sundays.
- [x] Pull requests build every changed image for `linux/amd64` and
  `linux/arm64` without publishing.
- [x] Shared build-workflow changes build all images.
- [x] Only main, release tags, and manual runs can publish or sign images.
- [ ] Confirm the live Renovate dashboard detects both the `dockerfile` and
  `github-actions` managers after enabling the GitHub App.
- [ ] Add image-specific smoke tests; a successful build does not prove that a
  daemon starts or its key protocol works.

### Supply chain and releases

- [x] Published images are signed keylessly with cosign.
- [x] Version tags are image-scoped, for example `samba-ad/v0.1.11`.
- [x] Deployments use versioned image tags rather than `:main` or `:latest`.
- [ ] Pin GitHub Actions to immutable commit SHAs and let Renovate maintain the
  human-readable version comments.
- [ ] Explicitly generate and attest an SBOM and provenance, then verify both
  and the cosign signature in at least one consumer or release check.
- [ ] Validate release tag syntax and require the image directory to contain
  the version being released where applicable.
- [ ] Define GHCR retention for mutable `:main`, build cache, and superseded
  development artifacts while retaining deployed release tags.

### Image construction and runtime

- [ ] Choose and document a base-image policy. Current release tags are
  readable but mutable; digest pinning is more reproducible but requires
  deliberate Renovate handling.
- [ ] Decide package-pinning policy per distribution. Exact pins improve
  reproducibility but can block security fixes; unpinned installs make rebuilds
  non-reproducible. Record intentional exceptions.
- [ ] Verify every downloaded signing key or standalone artifact against a
  documented fingerprint, checksum, or signature.
- [ ] Run as a non-root user where the daemon and mounted-file ownership allow
  it. Where root is required, document why and minimize runtime capabilities in
  the deployment repository.
- [ ] Remove unsafe example credential defaults and ensure shell tracing cannot
  print credentials.
- [ ] Add a minimal health/smoke test for each image before making its patch
  updates eligible for automerge.

## Current consumers

The `homeinfra` repository currently deploys:

| Image | Consumer |
|---|---|
| `asterisk:v20.11.1` | Asterisk chart |
| `aws-cli-tgz:v2.33.27` | Vaultwarden backup CronJob |
| `freeradius:v3.2.8` | FreeRADIUS chart |
| `samba-ad:v0.1.11` | Samba AD chart |

`dnsmasq`, `novnc`, and `openldap` have no current `homeinfra` consumer found
in this audit. Keep or retire them deliberately; do not assume that absence
from `homeinfra` proves they are unused everywhere.

## Per-image findings

| Image | Existing strengths | Deferred findings | Owner |
|---|---|---|---|
| `asterisk` | Small Alpine base; no external downloads | Alpine base is not on the current repository-wide version; packages are unpinned; runs as root; no startup/SIP smoke test | Future image-hardening stage |
| `aws-cli-tgz` | Versioned upstream base; narrow purpose | Added `tar`, `xz`, and `gzip` packages are unpinned; runs as root; no archive/CLI smoke test | Future image-hardening stage |
| `dnsmasq` | Exact Alpine and package versions; runs as `nobody` | No DNS startup/query test; exact distro package pin must move with Alpine repositories | Future image-hardening or retirement stage |
| `freeradius` | Dedicated service UID/GID exists; signed APT repository configuration | Downloaded repository key is not checked against a fingerprint; repository URL is HTTP after key bootstrap; packages are unpinned; effective runtime user depends on configuration; no RADIUS smoke test | Future FreeRADIUS image stage |
| `novnc` | Slim Debian base; no external artifact download | Floating Debian suite and unpinned packages; runs as root; default command only prints help; no WebSocket smoke test | Future image-hardening or retirement stage |
| `openldap` | Version family is constrained | Unsafe default admin password; `set -x` risks logging secret-bearing commands; wildcard package versions are not reproducible; root startup and shell robustness need review; no LDAP smoke test | High-priority future OpenLDAP stage or retirement |
| `samba-ad` | Versioned releases; current multi-architecture build succeeds | Unsafe default admin password, hard-coded DNS forwarder, runtime hostname and `/etc/hosts` mutation, swallowed policy errors, no AD smoke test, and privileged deployment | Stage 51 configuration/image refactor; Stage 52 privilege canary |

## Prioritized follow-up

1. Complete Stage 51 and Stage 52 for the deployed Samba AD image.
2. Remove the unused `openldap` image in the planned retirement stage. Decide
   separately whether `novnc` and `dnsmasq` should be retained.
3. Add protocol-level smoke tests for deployed images, beginning with
   FreeRADIUS and Asterisk.
4. Add immutable Action pinning, explicit SBOM/provenance attestations, and
   verification.
5. Document and enforce base/package, release-validation, and retention
   policies across the remaining images.
