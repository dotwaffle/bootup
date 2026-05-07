## ADDED Requirements

### Requirement: Generic Linux source artifact hash pins
Bootup SHALL allow generic Linux catalog targets to plan and verify
catalog-supplied SHA-256 pins for source kernel and initrd artifacts.

#### Scenario: Generic Linux kernel hash is planned
- **WHEN** a generic Linux catalog target includes a kernel SHA-256 source hash
- **THEN** the Linux provider SHALL place that hash on the planned kernel
  artifact

#### Scenario: Generic Linux initrd hash is planned
- **WHEN** a generic Linux catalog target includes an initrd source path and an
  initrd SHA-256 source hash
- **THEN** the Linux provider SHALL place that hash on the planned initrd
  artifact

#### Scenario: Generic Linux pinned artifacts are staged
- **WHEN** a generic Linux boot plan contains SHA-256 pins for its downloadable
  artifacts
- **THEN** staging SHALL verify each downloaded artifact against its planned
  hash before writing the artifact into the staging directory
