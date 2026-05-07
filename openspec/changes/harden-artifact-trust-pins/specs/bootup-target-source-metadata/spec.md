## ADDED Requirements

### Requirement: Target source artifact hash metadata
Bootup SHALL support optional SHA-256 hash metadata for catalog source
artifacts that are planned directly from source paths.

#### Scenario: Kernel hash metadata is present
- **WHEN** a static catalog target includes a kernel source path and a kernel
  SHA-256 source hash
- **THEN** bootup SHALL preserve that hash as target source metadata for
  provider planning

#### Scenario: Initrd hash metadata is present
- **WHEN** a static catalog target includes an initrd source path and an initrd
  SHA-256 source hash
- **THEN** bootup SHALL preserve that hash as target source metadata for
  provider planning

#### Scenario: Source hash metadata is malformed
- **WHEN** a static catalog target includes a kernel or initrd source hash that
  is not a 64-character SHA-256 hex digest
- **THEN** bootup SHALL reject the catalog before registering provider targets

#### Scenario: Initrd source hashes are partial
- **WHEN** a static catalog target has an initrd source path and supplies only
  one of the kernel or initrd source hashes
- **THEN** bootup SHALL reject the catalog before registering provider targets
