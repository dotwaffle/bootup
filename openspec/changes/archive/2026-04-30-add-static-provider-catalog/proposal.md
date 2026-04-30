## Why

Bootup currently exposes a flat list of provider targets with ad hoc metadata
fields. To scale toward a static embedded catalog of many concrete bootable
targets, providers need a stable catalog metadata contract that UIs and future
catalog tooling can consume without understanding provider internals.

## What Changes

- Add a typed catalog metadata model for static, concrete boot targets.
- Require registered provider targets to carry enough catalog metadata for
  grouping and presentation.
- Move current Debian and Ubuntu target metadata into that catalog model.
- Keep the current target list and menu behavior functionally unchanged.
- Document that dynamic release discovery and fully dynamic policy/scripted
  boot decisions are future modes, not part of this static catalog slice.

## Capabilities

### New Capabilities

- `bootup-static-provider-catalog`: static catalog metadata for compiled-in,
  concrete provider targets.

### Modified Capabilities

- `bootup-netboot`: build-time providers expose static catalog metadata through
  a typed target catalog contract.

## Impact

- Affects `internal/provider` target data structures and registry validation.
- Affects provider implementations and tests for Debian and Ubuntu target
  metadata.
- Affects text/rich UI metadata rendering through the new catalog field.
- Adds provider catalog documentation for the supported mode and deferred future
  modes.
