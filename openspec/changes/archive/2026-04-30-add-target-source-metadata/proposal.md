## Why

The static catalog can now list more targets, but some providers cannot derive
all source URLs from distribution, release, and architecture alone. Bootup needs
per-target source metadata before expanding the catalog with targets such as
Ubuntu point releases whose release URL and installer ISO name are concrete
target facts.

## What Changes

- Add optional per-target source metadata to static catalog targets.
- Validate target source metadata during catalog/provider target validation.
- Teach Debian planning to use a target source base URL when present, otherwise
  preserving provider runtime/default mirror behavior.
- Teach Ubuntu planning to use target source base URLs and per-target installer
  ISO names while preserving the existing 26.04 default behavior.
- Expand the embedded default catalog with Debian bullseye amd64 netboot and
  Ubuntu 24.04.4/25.10 amd64 netboot.
- Update docs, tests, and VM target-list coverage for the expanded static
  catalog.

## Capabilities

### New Capabilities
- `bootup-target-source-metadata`: Optional static catalog metadata describing
  provider source facts for a concrete target.

### Modified Capabilities
- `bootup-static-catalog-source`: Static catalog documents can carry validated
  per-target source metadata and the default catalog includes additional
  concrete targets.
- `bootup-netboot`: Debian and Ubuntu providers resolve additional default
  static catalog targets from per-target source metadata where needed.

## Impact

- Extends `provider.Target` with an optional source metadata block.
- Updates catalog JSON validation and embedded default catalog content.
- Updates Debian and Ubuntu provider planning tests and implementation.
- Preserves provider runtime trust configuration and existing default target
  behavior.
- Does not add hosted catalog loading, dynamic distro discovery, or new
  distribution trust material.
