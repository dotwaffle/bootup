# bootup-target-source-metadata Specification

## Purpose
Define optional static catalog source metadata for concrete provider targets,
including source URLs and provider-specific artifact names that are target
facts rather than executable provider behavior.
## Requirements
### Requirement: Target source metadata
Bootup SHALL support optional typed source metadata for concrete static catalog
targets.

#### Scenario: Target source metadata is absent
- **WHEN** a static catalog target omits source metadata
- **THEN** bootup SHALL preserve provider runtime configuration and provider
  default source behavior for that target

#### Scenario: Target source base URL is present
- **WHEN** a static catalog target includes a source base URL
- **THEN** the target provider SHALL use that base URL when planning artifacts
  for the selected target

#### Scenario: Target source ISO name is present
- **WHEN** a static catalog target includes an ISO name
- **THEN** the target provider SHALL use that ISO name for provider workflows
  that require a concrete installer ISO filename

### Requirement: Target source validation
Bootup SHALL validate target source metadata before provider target discovery.

#### Scenario: Source base URL is invalid
- **WHEN** a static catalog target includes a source base URL that is not an
  absolute HTTP or HTTPS URL with a host
- **THEN** bootup SHALL reject the catalog before registering provider targets

#### Scenario: Source ISO name is invalid
- **WHEN** a static catalog target includes an ISO name containing path
  separators or surrounding whitespace
- **THEN** bootup SHALL reject the catalog before registering provider targets

#### Scenario: Source metadata is data only
- **WHEN** bootup loads source metadata from a static catalog target
- **THEN** the metadata MUST NOT cause bootup to load provider code from the
  network or from runtime plugin files

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
