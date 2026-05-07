# bootup-release-packaging Specification

## Purpose
Define bootup's amd64 release artifact contract, manifest and checksum
metadata, validation gates, publication workflow, and operator-facing release
usage documentation.
## Requirements
### Requirement: Release artifact contract
Bootup SHALL define a stable amd64 release artifact set for operators and
automation.

#### Scenario: Release bundle is assembled
- **WHEN** the release packaging workflow builds an amd64 release
- **THEN** it SHALL produce a bootup binary, bootup kernel image, kernel config,
  zstd-compressed bootup initramfs, hybrid BIOS/UEFI ISO, checksum file, and
  machine-readable manifest

#### Scenario: Artifact names are stable
- **WHEN** release artifacts are written for publication
- **THEN** each public artifact name SHALL include the bootup release version
  and target architecture

#### Scenario: Kernel artifacts identify the kernel version
- **WHEN** release kernel artifacts are written for publication
- **THEN** the kernel image and kernel config names SHALL include the Linux
  kernel version used to build them

#### Scenario: Default artifacts exclude distribution trust bundles
- **WHEN** the default release initramfs and ISO are built
- **THEN** they MUST NOT embed distribution-specific archive keyrings or trust
  bundles

### Requirement: Release manifest and checksums
Bootup SHALL publish integrity metadata alongside release artifacts.

#### Scenario: Checksums are generated
- **WHEN** a release bundle is assembled
- **THEN** the release workflow SHALL generate a checksum file containing the
  SHA-256 digest for every public binary artifact and the manifest

#### Scenario: Manifest describes artifacts
- **WHEN** a release manifest is generated
- **THEN** it SHALL record the manifest schema version, bootup release version,
  git commit, target architecture, artifact roles, artifact names, byte sizes,
  and SHA-256 digests

#### Scenario: Manifest describes trust material posture
- **WHEN** a release manifest is generated for default artifacts
- **THEN** it SHALL state that distribution-specific trust material is not
  embedded in the release artifacts

### Requirement: Release validation gates
Bootup SHALL validate release artifacts before publication.

#### Scenario: Release validation succeeds
- **WHEN** a release publication job runs
- **THEN** it SHALL pass script syntax checks, Go tests, lint checks,
  manifest/checksum verification, artifact presence checks, and an ISO boot
  smoke before publishing release assets

#### Scenario: Release validation fails
- **WHEN** any required release validation step fails
- **THEN** the release workflow MUST fail before publishing or updating release
  assets

#### Scenario: ISO content is checked
- **WHEN** release validation inspects the hybrid ISO
- **THEN** it SHALL verify that the ISO contains the bootup kernel, bootup
  initramfs, GRUB configuration, and UEFI fallback boot path

### Requirement: Release publication workflow
Bootup SHALL provide an automated workflow for release publication.

#### Scenario: Tagged release is published
- **WHEN** a release tag is pushed
- **THEN** the release workflow SHALL build, validate, and publish the release
  artifact set for that tag

#### Scenario: Manual release build is available
- **WHEN** an operator starts the release workflow manually
- **THEN** the workflow SHALL build and validate release artifacts without
  requiring a source change

#### Scenario: Release permissions are scoped
- **WHEN** normal pull-request or branch CI runs
- **THEN** it MUST NOT require permissions to publish release assets

### Requirement: Release usage documentation
Bootup SHALL document how operators consume release artifacts.

#### Scenario: iPXE artifact usage is documented
- **WHEN** an operator wants to boot with iPXE
- **THEN** the release documentation SHALL identify the kernel and zstd
  initramfs artifacts and show the expected launch shape

#### Scenario: GRUB artifact usage is documented
- **WHEN** an operator wants to chainload bootup with GRUB
- **THEN** the release documentation SHALL identify the kernel and zstd
  initramfs artifacts and show the expected menu entry shape

#### Scenario: ISO artifact usage is documented
- **WHEN** an operator wants to boot from local media or virtual media
- **THEN** the release documentation SHALL identify the hybrid ISO artifact and
  describe BIOS and UEFI boot expectations

#### Scenario: Artifact verification is documented
- **WHEN** an operator downloads release artifacts
- **THEN** the release documentation SHALL show how to verify the checksum file
  and inspect the manifest before booting

### Requirement: Release binary build metadata
Bootup SHALL expose stamped build metadata from the release binary and publish
that metadata in the release manifest.

#### Scenario: Version command reports build metadata
- **WHEN** an operator runs `bootup --version`
- **THEN** bootup SHALL print the bootup release version, git commit, build
  date, source tree state, and Go runtime version without loading catalogs,
  provider configuration, or boot modes

#### Scenario: Release builder stamps binary metadata
- **WHEN** the release packaging workflow builds the standalone bootup binary
- **THEN** it SHALL stamp the binary with the release version, git commit,
  build date, and source tree state used for the release

#### Scenario: Manifest records binary metadata
- **WHEN** a release manifest is generated
- **THEN** it SHALL record the stamped bootup binary release version, git
  commit, build date, and source tree state

#### Scenario: Release validation checks binary metadata
- **WHEN** release validation inspects a release bundle
- **THEN** it SHALL compare the manifest's bootup binary metadata with the
  metadata reported by the release binary before publication
