## ADDED Requirements

### Requirement: Typed boot actions
Bootup SHALL represent the selected handoff method with a typed boot action on
targets and boot plans.

#### Scenario: Existing targets default to Linux kexec
- **WHEN** a target or boot plan omits a boot action
- **THEN** bootup SHALL treat it as a `linux-kexec` action

#### Scenario: Unknown action is rejected
- **WHEN** a target or boot plan declares an unsupported boot action
- **THEN** bootup SHALL reject it before attempting staging or handoff

### Requirement: Local disk boot action
Bootup SHALL support a local disk boot action that uses the bundled u-root boot
path instead of downloading remote artifacts.

#### Scenario: Local boot target is selected
- **WHEN** the operator selects the local disk boot target
- **THEN** bootup SHALL plan a `localboot` handoff without kernel or initrd
  artifacts

#### Scenario: Local boot executes
- **WHEN** bootup executes a staged `localboot` plan
- **THEN** it SHALL invoke the configured local boot command and report any
  command failure

### Requirement: Deferred chainload actions
Bootup SHALL not advertise BSD, memdisk ISO, syslinux COM32, or chainload
targets as executable until a dedicated executor family supports them.

#### Scenario: Unsupported target family is considered
- **WHEN** a candidate target requires memdisk, syslinux module, or chainload
  semantics
- **THEN** bootup SHALL keep that target out of the executable default catalog
  and document it as future work
