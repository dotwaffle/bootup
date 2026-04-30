## ADDED Requirements

### Requirement: Debian bookworm amd64 netboot target
Bootup SHALL include Debian bookworm amd64 netboot in the default static
catalog for the compiled-in Debian provider.

#### Scenario: Debian bookworm target is listed
- **WHEN** bootup starts with the Debian provider compiled in and the default
  static catalog selected
- **THEN** the operator interface SHALL offer Debian bookworm amd64 netboot as a
  selectable target

#### Scenario: Debian bookworm target carries catalog metadata
- **WHEN** bootup lists the Debian bookworm amd64 netboot target
- **THEN** the target SHALL include distribution, release, architecture, and
  target-kind metadata suitable for catalog grouping

#### Scenario: Debian provider resolves bookworm installer artifacts
- **WHEN** the operator selects Debian bookworm amd64 netboot
- **THEN** the provider SHALL resolve the Debian Installer kernel, initrd, and
  required kernel command line for amd64 netboot using bookworm release paths
