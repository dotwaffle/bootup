## Why

Bootup can now build the kernel, initramfs, and hybrid ISO locally, but those
outputs do not yet have a stable release contract or repeatable CI publication
path. Defining release packaging now prevents local `dist/` behavior from
becoming an accidental public API.

## What Changes

- Define the release artifact set and naming convention for bootup binaries,
  kernel images, initramfs images, hybrid ISOs, checksums, and manifests.
- Add release automation that builds the artifacts in CI and publishes them for
  tag-based releases.
- Add validation gates that check artifact contents, checksum/manifest
  integrity, script syntax, Go tests, lint, and at least one ISO boot smoke.
- Document which artifact to use for iPXE, GRUB, and ISO boot paths.
- Keep distribution trust material operator-configured: default release
  artifacts must not embed distribution-specific archive keyrings or trust
  bundles.

## Capabilities

### New Capabilities
- `bootup-release-packaging`: Release artifact layout, manifest/checksum
  contract, CI publication flow, validation gates, and operator-facing release
  documentation.

### Modified Capabilities
- None.

## Impact

- Affected code and scripts: release build scripts, existing kernel/initramfs/ISO
  helpers, validation helpers, and CI workflow files.
- Affected docs: release artifact usage, checksums/manifests, and stage-0 boot
  examples for iPXE, GRUB, and ISO media.
- Affected systems: GitHub Actions or equivalent CI release automation.
