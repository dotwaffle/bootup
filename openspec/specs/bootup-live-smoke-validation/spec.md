# bootup-live-smoke-validation Specification

## Purpose
Define opt-in live smoke validation for selected catalog targets, keeping
network and QEMU-dependent checks explicit while preserving hermetic default
tests.
## Requirements
### Requirement: Catalog target live smoke selection
Bootup SHALL provide an explicit opt-in live smoke path for selected static
catalog targets that can be exercised by currently implemented boot actions,
and SHALL use the catalog smoke coverage classification when deciding whether a
target is supported by the live catalog staging path.

#### Scenario: Supported target is smoke selectable
- **WHEN** a static catalog target is classified for live catalog staging smoke
  support
- **THEN** the live smoke path SHALL allow the target to be selected by target
  ID

#### Scenario: Unsupported action is skipped
- **WHEN** a static catalog target lacks live catalog staging smoke support,
  including targets that require memdisk, syslinux COM32, HDT, BSD-specific
  handoff, chainload, local boot, or a dedicated non-catalog smoke helper
- **THEN** the live smoke path SHALL report that the target is unsupported
  rather than attempting to boot it

### Requirement: Kernel and initrd live smoke coverage
Bootup SHALL include live smoke coverage for at least one static catalog target
that boots through Linux kexec with both kernel and initrd artifacts.

#### Scenario: Generic Linux target is smoke tested
- **WHEN** live smoke validation is explicitly enabled for a selected generic
  Linux static catalog target
- **THEN** bootup SHALL stage both kernel and initrd artifacts and attempt the
  configured VM boot path

### Requirement: Live smoke isolation
Bootup SHALL keep live smoke validation out of the default hermetic test suite.

#### Scenario: Default tests do not run live smoke
- **WHEN** the normal repository test suite is run without live smoke tags or
  environment variables
- **THEN** live smoke validation SHALL NOT contact upstream mirrors or launch
  QEMU
