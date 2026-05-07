## Why

Bootup keeps the failed stage-1 environment available, but failures are still
hard to inspect after a noisy serial session or QEMU smoke run. Operators need
an opt-in way to persist the selected mode, target, logs, output, and failure
summary without changing the normal console behavior.

## What Changes

- Add an opt-in diagnostics directory flag for writing failure reports.
- Capture bootup stdout, stderr/log output, mode, target inputs, discovery
  family, selected option IDs, catalog source posture, provider config path
  presence, and the final error.
- Write diagnostics as JSON plus text stream files so operators and tests can
  inspect the same failure state.
- Keep diagnostics best-effort: preserve the original boot error and report a
  diagnostics write failure without masking the boot failure.
- Document local and VM usage for diagnostics capture.

## Capabilities

### New Capabilities

- `bootup-persistent-diagnostics`: opt-in persistent diagnostics for failed
  bootup startup modes.

### Modified Capabilities

None.

## Impact

- Affected code: `cmd/bootup`, `internal/app` wiring or a small internal
  diagnostics package, docs, and tests.
- Public behavior: new `--diagnostics-dir` flag; no behavior change when the
  flag is absent.
- Dependencies: no new third-party dependencies.
