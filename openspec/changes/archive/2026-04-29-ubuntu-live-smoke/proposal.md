## Why

Ubuntu 26.04 is now listed and stageable by provider code, but it lacks a
repeatable live smoke path. A focused smoke helper and opt-in test will prove
the HTTPS staging path before we invest in richer catalog UI.

## What Changes

- Add a local QEMU smoke helper that builds an initramfs and attempts to kexec
  into Ubuntu 26.04 netboot.
- Add an opt-in live Ubuntu staging test that skips unless explicitly enabled.
- Document exact commands and the current HTTPS-only trust model.
- Keep default tests hermetic and skip live Ubuntu checks unless requested.

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `bootup-netboot`: Adds repeatable live smoke coverage for Ubuntu 26.04
  netboot staging and handoff attempts.

## Impact

- Adds a smoke script under `scripts/`.
- Extends `test/live` with Ubuntu coverage.
- Updates launch and VM test documentation.
- Does not add Ubuntu keyrings, generated hashes, or binary artifacts.
