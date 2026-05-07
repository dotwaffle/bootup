## Why

The static catalog now spans multiple providers and boot actions, but operators
and tests do not have a single stable view of which targets are metadata-only,
plan-checkable, live-stage smokeable, or QEMU smokeable. That makes hosted
catalog rollout and future provider additions harder to audit before staging
artifacts.

## What Changes

- Add a catalog conformance and smoke matrix that reports every registered
  target with provider, boot action, plan support, artifact-trust posture, and
  smoke coverage classification.
- Add an operator-facing mode for rendering the matrix without downloading
  artifacts, contacting upstream mirrors, or launching QEMU.
- Reuse the same support classification in live catalog smoke tests so
  unsupported targets are reported consistently.
- Document the matrix and keep network/QEMU smoke paths opt-in.

## Capabilities

### New Capabilities
- `bootup-catalog-conformance`: catalog target conformance reporting and smoke
  coverage classification.

### Modified Capabilities
- `bootup-live-smoke-validation`: live smoke selection SHALL use the catalog
  smoke classification and SHALL keep unsupported target reporting aligned with
  the matrix.

## Impact

- Affected code: `internal/catalog`, `internal/app`, `cmd/bootup`, `test/live`,
  and docs for VM/live smoke validation.
- APIs: new internal conformance report types and a new non-interactive
  startup mode.
- Dependencies: no new external dependencies.
- Systems: default tests remain hermetic; live staging and QEMU smokes remain
  explicit opt-in paths.
