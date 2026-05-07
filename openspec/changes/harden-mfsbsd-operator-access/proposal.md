## Why

Bootup can now reach an mfsBSD serial login, but the target is still thinly
described as a boot proof. Operators need clear runtime expectations and a
small non-secret customization path before using it as the supported BSD rescue
bridge.

## What Changes

- Make the mfsBSD `freebsd-kboot` handoff pass explicit runtime loader
  variables for auto-DHCP and hostname.
- Add a catalog target option that lets operators override the mfsBSD hostname
  without changing provider code.
- Apply selected target option fragments to `freebsd-kboot` loader arguments
  instead of silently ignoring them.
- Document the supported mfsBSD login, serial console, DHCP, SSH, and ZFS
  installer expectations.
- Keep root password customization out of scope until bootup has a secret-safe
  plan/stage output path or a file-based secret mechanism.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `bootup-freebsd-kboot-handoff`: require mfsBSD runtime arguments to carry
  operator-visible rescue settings and preserve selected non-secret target
  options through staging.
- `bootup-target-options`: allow selected target option fragments to apply to
  non-Linux boot actions with action-specific behavior.

## Impact

- Affects `internal/provider` option application, the mfsBSD provider, generated
  static catalog data, mfsBSD-focused docs, and package tests.
- No new external dependencies.
- No generated or downloaded FreeBSD/mfsBSD binaries are committed.
