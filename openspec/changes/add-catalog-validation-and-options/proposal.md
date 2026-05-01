## Why

The static catalog now contains generic Linux and utility targets, but it has
not yet proved those targets boot in a VM or exposed their operator-tunable
installer choices in a structured way. As the catalog grows, operators also
need list/show output that makes target action, distro, version, architecture,
and options easy to inspect before booting.

## What Changes

- Add explicit live smoke coverage for selected static catalog targets, starting
  with one kernel-only target and one kernel+initrd target.
- Add catalog-declared installer options that append validated command-line
  fragments without hard-coding distro-specific UI into providers.
- Improve catalog discovery commands so operators can inspect target metadata,
  boot action, provider, artifacts, and option sets.
- Keep BSD, memdisk, syslinux, HDT, and chainload execution out of scope except
  for documenting why unsupported actions are skipped by smoke coverage.

## Capabilities

### New Capabilities

- `bootup-live-smoke-validation`: VM smoke validation for catalog targets that
  can be exercised by the current executor set.
- `bootup-target-options`: catalog-declared installer and kernel command-line
  options that can be selected by operators.
- `bootup-catalog-discovery`: operator-facing catalog list/show behavior for
  inspecting targets before booting.

### Modified Capabilities

- `bootup-static-catalog-source`: source entries can declare structured target
  options.
- `bootup-static-provider-catalog`: generated catalogs preserve and expose target
  option metadata.

## Impact

- CLI catalog commands and output formatting.
- Static catalog source schema, generated catalog JSON, and validation.
- Provider planning path where selected options are translated into boot plan
  command-line additions.
- VM/live test scripts or tagged Go tests for current static targets.
