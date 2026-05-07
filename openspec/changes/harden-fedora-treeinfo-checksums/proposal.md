## Why

Fedora Server install trees publish `.treeinfo` checksums for the pxeboot
kernel and initrd, but bootup currently treats Fedora staging as HTTPS-only
unless operators preconfigure explicit hashes. Using Fedora's own tree metadata
gives default Fedora netboot staging a stronger integrity check without
embedding Fedora trust bundles or adding live-network test requirements.

## What Changes

- Preserve Fedora pxeboot SHA-256 pins in the embedded static catalog when the
  selected catalog target already carries source hash metadata.
- Fetch and parse Fedora install-tree `.treeinfo` metadata when Fedora netboot
  artifacts are planned without explicit runtime or target source hash pins.
- Populate the planned kernel and initrd SHA-256 fields from `.treeinfo` so the
  existing staging verifier checks both downloads before writing them.
- Fail closed when `.treeinfo` is unavailable, malformed, or missing the
  required pxeboot checksum entries.
- Keep explicit provider runtime hash pins authoritative; when both kernel and
  initrd pins are configured, planning continues to use those values without
  using catalog pins or fetching `.treeinfo`.
- Add hermetic provider and command tests plus docs for the Fedora metadata
  trust posture.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `bootup-netboot`: Fedora netboot planning uses install-tree `.treeinfo`
  checksums by default instead of relying on HTTPS-only artifact staging.

## Impact

- Affected code: `internal/providers/fedora`, command provider registration
  tests if needed, docs, and OpenSpec netboot requirements.
- Public behavior: Fedora planning may now fail before staging when Fedora
  `.treeinfo` cannot provide checksums and explicit runtime pins are absent.
- Dependencies: no new third-party dependencies.
