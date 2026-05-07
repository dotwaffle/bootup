## ADDED Requirements

### Requirement: FreeBSD kboot smoke proves kernel metadata prerequisites
Bootup SHALL require an opt-in QEMU smoke before treating the FreeBSD kboot
handoff path as viable, and that smoke SHALL use a bootup kernel that exposes
the Linux metadata interfaces required by FreeBSD `loader.kboot`.

#### Scenario: Smoke includes kernel prerequisite evidence
- **WHEN** the FreeBSD kboot smoke is run
- **THEN** it SHALL verify or document that the bootup kernel exposes
  `CONFIG_KALLSYMS`, `CONFIG_KALLSYMS_ALL`, and `CONFIG_PROC_KCORE`

#### Scenario: Smoke uses non-vendored FreeBSD artifacts
- **WHEN** the FreeBSD kboot smoke stages `loader.kboot` and a FreeBSD or
  mfsBSD payload
- **THEN** those artifacts SHALL be downloaded or supplied by path outside the
  tracked repository and SHALL NOT be committed

#### Scenario: Smoke success signal reaches target environment
- **WHEN** the FreeBSD kboot handoff is reported as viable
- **THEN** the smoke evidence SHALL show serial output reaching the FreeBSD
  installer or an mfsBSD shell, not merely the FreeBSD loader menu

#### Scenario: Smoke failure keeps targets deferred
- **WHEN** the FreeBSD kboot smoke fails before reaching the FreeBSD installer
  or an mfsBSD shell
- **THEN** bootup SHALL keep FreeBSD and mfsBSD targets out of the executable
  default catalog and record the next blocking condition
