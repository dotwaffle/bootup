## ADDED Requirements

### Requirement: Debian bullseye amd64 netboot target
Bootup SHALL include Debian bullseye amd64 netboot in the default static
catalog for the compiled-in Debian provider.

#### Scenario: Debian bullseye target is listed
- **WHEN** bootup starts with the Debian provider compiled in and the default
  static catalog selected
- **THEN** the operator interface SHALL offer Debian bullseye amd64 netboot as a
  selectable target

#### Scenario: Debian provider resolves bullseye installer artifacts
- **WHEN** the operator selects Debian bullseye amd64 netboot
- **THEN** the provider SHALL resolve the Debian Installer kernel, initrd, and
  required kernel command line for amd64 netboot using bullseye release paths

### Requirement: Ubuntu sourceful static catalog targets
Bootup SHALL allow the compiled-in Ubuntu provider to plan static catalog
targets whose release URL and installer ISO name are supplied by target source
metadata.

#### Scenario: Ubuntu 24.04.4 target is listed
- **WHEN** bootup starts with the default static catalog selected
- **THEN** the operator interface SHALL offer Ubuntu 24.04.4 amd64 netboot as a
  selectable target

#### Scenario: Ubuntu 25.10 target is listed
- **WHEN** bootup starts with the default static catalog selected
- **THEN** the operator interface SHALL offer Ubuntu 25.10 amd64 netboot as a
  selectable target

#### Scenario: Ubuntu target source resolves release artifacts
- **WHEN** the operator selects an Ubuntu static target with source metadata
- **THEN** the provider SHALL resolve netboot kernel, initrd, checksum, signature,
  and installer ISO URLs from that target's source metadata
