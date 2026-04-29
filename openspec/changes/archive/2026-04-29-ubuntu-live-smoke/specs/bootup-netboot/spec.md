## MODIFIED Requirements

### Requirement: VM-based boot verification
Bootup SHALL include integration coverage that exercises the stage-1 environment
and provider paths under a virtual machine.

#### Scenario: Real Ubuntu smoke is explicitly enabled
- **WHEN** the operator provides QEMU, local kernel/initramfs inputs, and
  network access
- **THEN** bootup SHALL provide a repeatable smoke path that attempts to stage
  live Ubuntu 26.04 netboot artifacts and kexec into the installer

#### Scenario: Real Ubuntu smoke inputs are absent
- **WHEN** required live-smoke inputs are missing
- **THEN** the Ubuntu smoke test SHALL skip without failing the default test
  suite
