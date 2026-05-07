## Why

The catalog matrix makes `https-only` targets visible, but generic Linux
catalog targets do not yet have a way for local or hosted catalogs to pin their
kernel and initrd artifacts. Adding validated catalog hash pins gives operators
a data-only path to harden those targets without embedding new provider code.

## What Changes

- Extend static target source metadata with optional SHA-256 pins for generic
  Linux kernel and initrd artifacts.
- Validate source hash pins before provider registration and reject malformed or
  partial kernel/initrd pin sets.
- Preserve source hash pins through generated embedded catalog output.
- Pass validated pins into generic Linux boot plans so staging verifies pinned
  artifacts and the catalog matrix reports `hash-pinned`.

## Capabilities

### New Capabilities

### Modified Capabilities
- `bootup-target-source-metadata`: add validated kernel/initrd SHA-256 source
  metadata for generic Linux-style artifact planning.
- `bootup-static-catalog-source`: preserve validated source artifact hash pins
  through generated static catalogs.
- `bootup-netboot`: require generic Linux provider staging to verify pinned
  catalog artifacts when pins are present.

## Impact

- Affected code: `internal/provider`, `internal/catalog`,
  `internal/providers/linux`, docs, and tests.
- APIs: `provider.SourceEntry` gains optional `kernel_sha256` and
  `initrd_sha256` JSON fields.
- Dependencies: no new external dependencies.
- Systems: default catalog targets remain HTTPS-only until pins are added;
  local and hosted catalog replacements can opt into hash-pinned generic Linux
  artifacts.
