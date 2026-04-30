## Why

The rich operator UI is now implemented, but it needs real terminal coverage
and boot-environment validation before further visual polish. We also need to
know the runtime size cost of the TUI dependencies.

## What Changes

- Add PTY-backed rich menu coverage that exercises real terminal input/output.
- Improve the target picker styling with stronger grouping, selected-row
  treatment, and an animated bootup banner.
- Measure bootup binary and initramfs sizes after the rich UI dependency
  promotion.
- Run QEMU menu smoke coverage for `--ui=auto` and document the observed
  behavior.

## Capabilities

### New Capabilities

### Modified Capabilities

- `bootup-netboot`: rich operator UI verification gains PTY and QEMU coverage,
  plus size measurements for the boot image.

## Impact

- Affects `internal/ui`, `internal/app` tests, docs, and local validation
  notes.
- Does not change provider APIs, artifact verification, staging, or kexec
  handoff semantics.
