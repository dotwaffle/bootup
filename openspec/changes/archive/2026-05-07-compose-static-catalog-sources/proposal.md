## Why

Local and hosted static catalogs currently replace the embedded default catalog.
That forces operators to copy the full default catalog when they only need to
add a site-local target.

## What Changes

- Add an opt-in catalog composition mode that combines the embedded default
  catalog with one selected local or hosted catalog source.
- Keep existing `--catalog` and `--catalog-url` replacement behavior unchanged
  unless composition is explicitly requested.
- Reject duplicate target IDs across composed catalog sources instead of
  defining override or shadowing behavior.
- Document the composition mode and add catalog/CLI tests.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `bootup-static-catalog-source`: static catalog source selection can
  explicitly compose a selected local or hosted catalog with the embedded
  default catalog.

## Impact

- Affected code: `internal/catalog`, `cmd/bootup`, docs, tests, and OpenSpec
  specs.
- Public behavior: new `--catalog-include-default` CLI flag.
- Dependencies: no new third-party dependencies.
