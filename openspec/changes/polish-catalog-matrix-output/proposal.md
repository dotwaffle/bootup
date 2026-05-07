## Why

The catalog matrix is useful, but operators still have to correlate target IDs
with catalog metadata by eye, and the hermetic dry-run contract needs to remain
explicit as providers gain normal-planning metadata fetches. This change makes
the matrix easier to scan while preserving its no-network role.

## What Changes

- Add catalog identity columns to the matrix output: distribution, release,
  architecture, kind, and lifecycle status.
- Make catalog matrix planning request offline metadata behavior so providers
  do not fetch remote metadata while rendering the matrix.
- Teach Fedora planning to return a deterministic offline planning error when
  a Fedora target needs `.treeinfo` and no runtime or target source pins are
  available.
- Update tests and docs for the expanded TSV header and offline matrix
  planning behavior.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `bootup-catalog-conformance`: matrix rows include catalog identity metadata
  and the matrix dry-run explicitly prohibits remote metadata fetches.

## Impact

- Affected code: `provider.PlanInput`, `internal/catalog`,
  `internal/providers/fedora`, `internal/app`, docs, and tests.
- Public behavior: `bootup --mode=catalog-matrix` TSV gains additional columns.
- Dependencies: no new third-party dependencies.
