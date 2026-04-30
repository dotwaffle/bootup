## Why

Static boot targets are currently compiled directly into provider code, which
does not scale to a larger mode-1 catalog or to an operator-supplied static
catalog. Bootup needs a data-backed catalog source before adding more concrete
targets and before designing dynamic provider discovery.

## What Changes

- Add an embedded default static provider catalog document for concrete boot
  targets.
- Add a local `--catalog` JSON file option that replaces the embedded default
  catalog at startup.
- Validate static catalog documents before registering targets with compiled-in
  providers.
- Expand the default static catalog with Debian bookworm amd64 netboot while
  preserving Debian trixie and Ubuntu 26.04 amd64 netboot.
- Generalize the Debian provider so configured static Debian targets can drive
  release-specific artifact planning.
- Document URL-hosted catalogs and dynamic distro discovery as future design
  boundaries, not implemented runtime behavior.

## Capabilities

### New Capabilities
- `bootup-static-catalog-source`: Static catalog documents used to source
  concrete provider targets.

### Modified Capabilities
- `bootup-static-provider-catalog`: Static catalog targets are sourced from an
  embedded or local catalog document instead of only provider constructors.
- `bootup-netboot`: The default provider set includes Debian bookworm amd64
  netboot in addition to the existing default targets.

## Impact

- Adds an internal catalog loader and embedded JSON catalog.
- Adds a `--catalog` command-line flag.
- Adjusts provider registration to pass catalog targets into compiled-in
  providers.
- Updates Debian provider planning to use target catalog release metadata.
- Adds docs and tests for local catalog loading and the dynamic discovery
  boundary.
