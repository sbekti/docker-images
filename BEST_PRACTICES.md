# Container image best-practices audit

This is the repository checklist and findings inventory as of 2026-08-08.
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
- [x] The live Renovate dashboard detects both the `dockerfile` and
  `github-actions` managers.

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

## Current consumers

The `homeinfra` repository currently deploys:

| Image | Consumer |
|---|---|
| `actual-cli:v26.8.0` | Actual Budget bank-sync CronJob |
| `asterisk:v20.11.1` | Asterisk chart |
| `freeradius:v3.2.8` | FreeRADIUS chart |
| `hermes-tools:v0.1.0` | Hermes tools sidecar |
| `radius-proxy:v1.1.0` | JFK1 RouterOS RADIUS proxy container |
| `samba-ad:v0.3.2` | Samba AD chart |

Vaultwarden no longer consumes `aws-cli-tgz`; its CronJob uses the external
`ttionya/vaultwarden-backup:1.27.0` image.

## Per-image findings

| Image | Existing strengths | Deferred findings | Owner |
|---|---|---|---|
| `actual-cli` | Dedicated image, non-root runtime, versioned upstream CLI package | Node base and npm dependency are mutable; no explicit package integrity check | Future image-hardening stage |
| `asterisk` | Small Alpine base; no external downloads | Alpine base is not on the current repository-wide version; packages are unpinned; runs as root | Future image-hardening stage |
| `aws-cli-tgz` | Versioned upstream base; narrow purpose | No current `homeinfra` consumer; added packages are unpinned; runs as root | Candidate for deprecation |
| `freeradius` | Dedicated service UID/GID exists; signed APT repository configuration | Downloaded repository key is not checked against a fingerprint; repository URL is HTTP after key bootstrap; packages are unpinned; effective runtime user depends on configuration | Future FreeRADIUS image stage |
| `hermes-tools` | Adds no packages; build-time Python compile/import validation; non-root runtime; no private deployment identifiers | Inherits the large upstream Hermes image and its mutable release tag | Stage 53 adoption; repository-wide base-image policy |
| `radius-proxy` | Reuses the signed FreeRADIUS base; versioned authentication/accounting behavior | Runtime remains coupled to the FreeRADIUS base and RouterOS deployment | Future image-hardening stage |
| `samba-ad` | Versioned releases, multi-architecture build, required configuration validation, `testparm` before startup, and reduced deployment capabilities | Ubuntu packages are unpinned; runtime remains root | Future package/runtime hardening |

## Prioritized follow-up

1. Add immutable Action pinning, explicit SBOM/provenance attestations, and
   verification.
2. Document and enforce base/package, release-validation, and retention
   policies across the remaining images.
