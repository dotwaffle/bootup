## ADDED Requirements

### Requirement: Chainloaded stage-1 environment
Bootup SHALL run as a Linux/u-root stage-1 environment loaded by an external
stage-0 boot path such as PXE, iPXE, GRUB, or ISO media.

#### Scenario: Bootup starts after stage-0 load
- **WHEN** a supported stage-0 boot path loads the bootup kernel and initramfs
- **THEN** bootup SHALL start inside Linux and take responsibility for target
  selection, artifact retrieval, verification, and handoff

#### Scenario: Bootup preserves launch-path independence
- **WHEN** bootup is launched from a different supported stage-0 path
- **THEN** bootup SHALL expose the same provider and target selection behavior
  after startup

### Requirement: Build-time provider modules
Bootup SHALL support operating system providers compiled into the distributed
image at build time.

#### Scenario: Provider is available in the image
- **WHEN** bootup starts with a provider included at build time
- **THEN** bootup SHALL list that provider's supported boot targets through the
  operator interface

#### Scenario: Runtime provider loading is absent
- **WHEN** bootup is running in the target environment
- **THEN** bootup MUST NOT require loading provider code from the network or
  from runtime plugin files

### Requirement: Provider boot planning
Each provider SHALL produce a boot plan describing the selected target,
required artifacts, command line, verification material, and staging behavior.

#### Scenario: Operator selects a target
- **WHEN** the operator selects a provider target
- **THEN** bootup SHALL request a boot plan from the provider before downloading
  target boot artifacts

#### Scenario: Provider cannot produce a valid plan
- **WHEN** the provider cannot resolve required artifacts or trust material
- **THEN** bootup SHALL report the planning failure and MUST NOT attempt kexec

### Requirement: Debian trixie amd64 netboot provider
Bootup SHALL include an MVP provider for Debian trixie amd64 netboot
installation.

#### Scenario: Debian trixie target is listed
- **WHEN** bootup starts with the Debian provider compiled in
- **THEN** the operator interface SHALL offer Debian trixie amd64 netboot as a
  selectable target

#### Scenario: Debian provider resolves installer artifacts
- **WHEN** the operator selects Debian trixie amd64 netboot
- **THEN** the provider SHALL resolve the Debian Installer kernel, initrd, and
  required kernel command line for amd64 netboot

### Requirement: Verified artifact chain
Bootup SHALL verify downloaded target boot artifacts before staging them for
kexec.

#### Scenario: Debian metadata verifies successfully
- **WHEN** the Debian provider downloads archive metadata and installer
  checksum data
- **THEN** bootup SHALL validate the signed metadata against explicitly
  configured Debian archive trust material before trusting installer checksums

#### Scenario: Debian trust material is absent
- **WHEN** the selected Debian provider has no configured archive trust
  material
- **THEN** bootup SHALL fail closed before staging artifacts and MUST NOT
  execute kexec

#### Scenario: Artifact checksum matches trusted metadata
- **WHEN** bootup downloads the selected Debian Installer kernel and initrd
- **THEN** bootup SHALL verify each artifact against trusted checksum metadata
  before staging it

#### Scenario: Verification fails
- **WHEN** signature validation or artifact checksum validation fails
- **THEN** bootup SHALL fail closed, report the verification error, and MUST NOT
  execute kexec

### Requirement: Network and time preparation
Bootup SHALL prepare enough network and time state to perform trusted remote
artifact retrieval.

#### Scenario: Network comes up
- **WHEN** bootup starts on a networked machine
- **THEN** bootup SHALL attempt to configure networking and DNS before provider
  discovery or artifact retrieval

#### Scenario: Time is not trustworthy
- **WHEN** bootup detects that system time is unsuitable for TLS or signature
  validation workflows
- **THEN** bootup SHALL attempt a configured time synchronization path before
  continuing network artifact retrieval

### Requirement: Serial-first operator interface
Bootup SHALL provide a plain operator interface suitable for serial consoles,
IPMI consoles, and console-like KVM sessions.

#### Scenario: Serial console is available
- **WHEN** bootup starts with a serial console
- **THEN** bootup SHALL render target selection and failure messages in a
  text-mode interface that remains usable in an 80x25 viewport

#### Scenario: Framebuffer is unavailable
- **WHEN** bootup cannot use a framebuffer display
- **THEN** bootup SHALL still allow target selection and boot execution through
  the plain operator interface

### Requirement: Kexec handoff
Bootup SHALL hand off to the selected target by staging verified artifacts and
executing kexec.

#### Scenario: Verified plan is ready
- **WHEN** all selected target artifacts are downloaded, verified, and staged
- **THEN** bootup SHALL invoke the configured kexec path with the provider's
  kernel, initrd, and command line

#### Scenario: Kexec is unavailable
- **WHEN** the running kernel or platform does not permit kexec
- **THEN** bootup SHALL report the failure clearly and MUST NOT discard the
  diagnostic state before the operator can inspect it

### Requirement: VM-based boot verification
Bootup SHALL include integration coverage that exercises the stage-1 environment
and Debian provider path under a virtual machine.

#### Scenario: VM test boots bootup
- **WHEN** the integration test launches bootup with QEMU/vmtest
- **THEN** the test SHALL observe bootup reaching the operator interface or an
  automated provider selection point

#### Scenario: VM test verifies Debian handoff preparation
- **WHEN** the integration test selects Debian trixie amd64 netboot
- **THEN** the test SHALL verify that bootup resolves, verifies, and stages the
  Debian Installer boot artifacts before kexec handoff using hermetic fixture
  metadata and trust material
