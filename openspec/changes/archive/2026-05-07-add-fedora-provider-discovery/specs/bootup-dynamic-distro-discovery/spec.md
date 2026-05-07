## ADDED Requirements

### Requirement: Fedora dynamic discovery
Bootup SHALL allow the compiled-in Fedora provider to discover amd64 Server
netboot targets from a configured Fedora releases index.

#### Scenario: Fedora family is listed
- **WHEN** the Fedora provider is compiled into bootup
- **THEN** bootup SHALL expose a Fedora discovery family without running
  discovery

#### Scenario: Fedora release is discovered
- **WHEN** Fedora discovery finds a numeric release with Server x86_64 netboot
  kernel and initrd paths
- **THEN** bootup SHALL return a concrete Fedora target with source base URL,
  release, architecture, and target kind metadata

#### Scenario: Fedora discovered target plans normally
- **WHEN** the operator selects a Fedora discovered target
- **THEN** the Fedora provider SHALL plan kernel, initrd, and install-tree URLs
  through the normal provider planning path
