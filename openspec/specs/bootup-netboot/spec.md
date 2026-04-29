# bootup-netboot Specification

## Purpose
Define bootup's chainloaded stage-1 netboot environment, provider model,
verified artifact staging, and kexec handoff behavior.

## Requirements
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

#### Scenario: Ubuntu provider is available in the image
- **WHEN** bootup starts with the default provider set
- **THEN** bootup SHALL list Ubuntu 26.04 amd64 netboot as a selectable target

#### Scenario: Catalog metadata is available
- **WHEN** a provider exposes targets
- **THEN** each target SHOULD include catalog metadata that allows future UIs to
  group entries without loading runtime plugins

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

#### Scenario: Debian trixie target carries catalog metadata
- **WHEN** bootup lists the Debian trixie amd64 netboot target
- **THEN** the target SHALL include distribution, release, architecture, and
  target-kind metadata suitable for catalog grouping

#### Scenario: Debian provider resolves installer artifacts
- **WHEN** the operator selects Debian trixie amd64 netboot
- **THEN** the provider SHALL resolve the Debian Installer kernel, initrd, and
  required kernel command line for amd64 netboot

### Requirement: Verified artifact chain
Bootup SHALL verify downloaded target boot artifacts before staging them for
kexec when provider verification material is available, and SHALL otherwise
constrain explicitly documented HTTPS-only provider paths to HTTPS URLs.

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

#### Scenario: Ubuntu netboot hashes are absent
- **WHEN** the selected Ubuntu provider lacks explicit netboot kernel and
  initrd hashes
- **THEN** bootup SHALL stage Ubuntu netboot artifacts only from HTTPS URLs

#### Scenario: Ubuntu netboot hashes are present
- **WHEN** the Ubuntu provider has release signing trust material and explicit
  netboot kernel and initrd hashes
- **THEN** bootup SHALL verify the signed release checksum file and each
  downloaded netboot artifact before staging it

### Requirement: Network and time preparation
Bootup SHALL prepare enough network and time state to perform trusted remote
artifact retrieval.

#### Scenario: Network is already configured
- **WHEN** bootup starts with kernel or loader-provided network configuration
- **THEN** bootup SHALL validate that a non-loopback network interface is
  configured before provider discovery or artifact retrieval

#### Scenario: Kernel DNS hints are available
- **WHEN** the kernel provides DNS hints through `/proc/net/pnp`
- **THEN** bootup SHALL make those hints available to normal resolver lookup
  when no resolver configuration exists

#### Scenario: Time is not trustworthy
- **WHEN** bootup detects that system time is unsuitable for TLS or signature
  validation workflows
- **THEN** bootup SHALL attempt a configured time synchronization path before
  continuing network artifact retrieval

### Requirement: Serial-first operator interface
Bootup SHALL provide an operator interface suitable for serial consoles, IPMI
consoles, and console-like KVM sessions.

#### Scenario: Serial console is available
- **WHEN** bootup starts with a serial console
- **THEN** bootup SHALL render target selection and failure messages in a
  interface that remains usable in an 80x25 viewport

#### Scenario: Rich terminal is available
- **WHEN** bootup menu mode starts with interactive terminal input and output
- **THEN** bootup SHALL render a bold target picker with color, keyboard
  navigation, animated activity, and selected-target detail

#### Scenario: Rich terminal is unavailable
- **WHEN** bootup menu mode starts with redirected input, redirected output, or
  an incompatible terminal
- **THEN** bootup SHALL fall back to the plain text target prompt

#### Scenario: Operator chooses a target by index
- **WHEN** bootup renders the serial menu
- **THEN** each target SHALL have a stable numeric index for selection

#### Scenario: Operator navigates rich menu
- **WHEN** bootup renders the rich terminal menu
- **THEN** arrow keys, `j`, and `k` SHALL move the highlighted target and
  Enter SHALL select it

#### Scenario: Boot progress is visible
- **WHEN** bootup is planning, staging, verifying, or loading a selected target
- **THEN** the serial interface SHALL render a concise status message that fits
  inside an 80-column viewport

#### Scenario: Rich boot progress is visible
- **WHEN** bootup is planning, staging, verifying, or loading a selected target
  after rich-menu selection
- **THEN** bootup SHALL render visually distinct progress status with animated
  activity before handoff

#### Scenario: Boot failure is visible
- **WHEN** planning, verification, staging, or kexec fails
- **THEN** the serial interface SHALL render a readable fatal error and keep the
  current environment available for diagnosis

#### Scenario: Framebuffer is unavailable
- **WHEN** bootup cannot use a framebuffer display
- **THEN** bootup SHALL still allow target selection and boot execution through
  the operator interface

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
