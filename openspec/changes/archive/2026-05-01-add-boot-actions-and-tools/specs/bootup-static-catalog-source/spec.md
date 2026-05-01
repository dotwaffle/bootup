## ADDED Requirements

### Requirement: Static Linux source metadata
Bootup SHALL allow static catalog targets for the generic Linux provider to
describe kernel path, optional initrd path, and command line source metadata.

#### Scenario: Generic Linux source target is loaded
- **WHEN** a catalog target references the generic Linux provider
- **THEN** bootup SHALL validate source base URL, kernel path, optional initrd
  path, and command line metadata before registering the target

### Requirement: Extended default utility targets
Bootup SHALL include Linux-shaped utility and installer targets in the embedded
default catalog.

#### Scenario: Local boot target is in default catalog
- **WHEN** bootup starts with the default static catalog
- **THEN** it SHALL expose a local disk boot target

#### Scenario: openSUSE target is in default catalog
- **WHEN** bootup starts with the default static catalog
- **THEN** it SHALL expose an openSUSE Leap amd64 installer target

#### Scenario: Arch Linux target is in default catalog
- **WHEN** bootup starts with the default static catalog
- **THEN** it SHALL expose an Arch Linux amd64 netboot target

#### Scenario: GParted target is in default catalog
- **WHEN** bootup starts with the default static catalog
- **THEN** it SHALL expose a GParted Live amd64 target

#### Scenario: MemTest86+ target is in default catalog
- **WHEN** bootup starts with the default static catalog
- **THEN** it SHALL expose a MemTest86+ amd64 target that uses the Linux kexec
  action without an initrd
