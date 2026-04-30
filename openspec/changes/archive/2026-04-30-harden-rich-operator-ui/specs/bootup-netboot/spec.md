## MODIFIED Requirements

### Requirement: VM-based boot verification
Bootup SHALL include integration coverage that exercises the stage-1 environment
and provider paths under a virtual machine.

#### Scenario: VM test boots bootup
- **WHEN** the integration test launches bootup with QEMU/vmtest
- **THEN** the test SHALL observe bootup reaching the operator interface or an
  automated provider selection point

#### Scenario: VM test verifies Debian handoff preparation
- **WHEN** the integration test selects Debian trixie amd64 netboot
- **THEN** the test SHALL verify that bootup resolves, verifies, and stages the
  Debian Installer boot artifacts before kexec handoff using hermetic fixture
  metadata and trust material

#### Scenario: Rich terminal menu is PTY tested
- **WHEN** the integration test launches menu mode with a pseudo-terminal
- **THEN** the test SHALL drive keyboard input through the terminal and observe
  the selected target being planned for boot

#### Scenario: Menu smoke runs under QEMU
- **WHEN** a local smoke run builds a menu-mode initramfs and launches QEMU
- **THEN** bootup SHALL reach the operator menu and accept serial keyboard
  selection in the VM

#### Scenario: Real Debian smoke is explicitly enabled
- **WHEN** the operator provides QEMU, local kernel/initramfs inputs, network
  access, and local Debian archive trust material
- **THEN** bootup SHALL provide a repeatable smoke path that attempts to stage
  live Debian Installer artifacts and kexec into the installer

#### Scenario: Real Debian smoke inputs are absent
- **WHEN** required live-smoke inputs are missing
- **THEN** the smoke test SHALL skip without failing the default test suite

#### Scenario: Real Ubuntu smoke is explicitly enabled
- **WHEN** the operator provides QEMU, local kernel/initramfs inputs, and
  network access
- **THEN** bootup SHALL provide a repeatable smoke path that attempts to stage
  live Ubuntu 26.04 netboot artifacts and kexec into the installer

#### Scenario: Real Ubuntu smoke inputs are absent
- **WHEN** required live-smoke inputs are missing
- **THEN** the Ubuntu smoke test SHALL skip without failing the default test
  suite
