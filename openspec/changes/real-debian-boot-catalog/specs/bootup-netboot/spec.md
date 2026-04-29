## MODIFIED Requirements

### Requirement: Debian trixie amd64 netboot provider
Bootup SHALL include an MVP provider for Debian trixie amd64 netboot
installation.

#### Scenario: Debian trixie target carries catalog metadata
- **WHEN** bootup lists the Debian trixie amd64 netboot target
- **THEN** the target SHALL include distribution, release, architecture, and
  target-kind metadata suitable for catalog grouping

### Requirement: Serial-first operator interface
Bootup SHALL provide a plain operator interface suitable for serial consoles,
IPMI consoles, and console-like KVM sessions.

#### Scenario: Operator chooses a target by index
- **WHEN** bootup renders the serial menu
- **THEN** each target SHALL have a stable numeric index for selection

#### Scenario: Boot progress is visible
- **WHEN** bootup is planning, staging, verifying, or loading a selected target
- **THEN** the serial interface SHALL render a concise status message that fits
  inside an 80-column viewport

#### Scenario: Boot failure is visible
- **WHEN** planning, verification, staging, or kexec fails
- **THEN** the serial interface SHALL render a readable fatal error and keep the
  current environment available for diagnosis

### Requirement: VM-based boot verification
Bootup SHALL include integration coverage that exercises the stage-1 environment
and Debian provider path under a virtual machine.

#### Scenario: Real Debian smoke is explicitly enabled
- **WHEN** the operator provides QEMU, local kernel/initramfs inputs, network
  access, and local Debian archive trust material
- **THEN** bootup SHALL provide a repeatable smoke path that attempts to stage
  live Debian Installer artifacts and kexec into the installer

#### Scenario: Real Debian smoke inputs are absent
- **WHEN** required live-smoke inputs are missing
- **THEN** the smoke test SHALL skip without failing the default test suite

### Requirement: Build-time provider modules
Bootup SHALL support operating system providers compiled into the distributed
image at build time.

#### Scenario: Catalog metadata is available
- **WHEN** a provider exposes targets
- **THEN** each target SHOULD include catalog metadata that allows future UIs to
  group entries without loading runtime plugins
