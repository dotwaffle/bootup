## ADDED Requirements

### Requirement: Fedora Server amd64 netboot provider
Bootup SHALL include a compiled-in Fedora Server amd64 netboot provider.

#### Scenario: Fedora static targets are listed
- **WHEN** bootup starts with the default provider set
- **THEN** the operator interface SHALL offer Fedora Server amd64 netboot
  targets from the embedded static catalog

#### Scenario: Fedora target carries source metadata
- **WHEN** bootup lists a Fedora Server static target
- **THEN** the target SHALL include source base URL metadata for the Fedora
  Server install tree

#### Scenario: Fedora provider resolves netboot artifacts
- **WHEN** the operator selects a Fedora Server amd64 netboot target
- **THEN** the provider SHALL resolve `images/pxeboot/vmlinuz`,
  `images/pxeboot/initrd.img`, and an `inst.repo=` kernel command line from the
  target source base URL

#### Scenario: Fedora hashes are absent
- **WHEN** the Fedora provider lacks explicit kernel and initrd hashes
- **THEN** bootup SHALL stage Fedora netboot artifacts only from HTTPS URLs

#### Scenario: Fedora hashes are present
- **WHEN** the Fedora provider has explicit kernel and initrd hashes
- **THEN** bootup SHALL verify each downloaded Fedora netboot artifact before
  staging it
