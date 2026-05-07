## MODIFIED Requirements

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
- **WHEN** the Fedora provider lacks explicit runtime and target source kernel
  and initrd hashes
- **THEN** bootup SHALL fetch Fedora install-tree `.treeinfo` metadata, require
  SHA-256 checksums for `images/pxeboot/vmlinuz` and
  `images/pxeboot/initrd.img`, and place those checksums in the boot plan

#### Scenario: Fedora treeinfo is incomplete
- **WHEN** the Fedora provider lacks explicit runtime and target source kernel
  and initrd hashes and the install-tree `.treeinfo` metadata is unavailable,
  malformed, or missing either pxeboot SHA-256 checksum
- **THEN** bootup SHALL fail planning before staging Fedora artifacts

#### Scenario: Fedora target source hashes are present
- **WHEN** the selected Fedora catalog target has explicit source kernel and
  initrd hashes and the provider lacks runtime hashes
- **THEN** bootup SHALL place the target source hashes in the boot plan without
  requiring Fedora `.treeinfo` metadata

#### Scenario: Fedora hashes are present
- **WHEN** the Fedora provider has explicit runtime kernel and initrd hashes
- **THEN** bootup SHALL verify each downloaded Fedora netboot artifact before
  staging it without using target source hashes or requiring Fedora `.treeinfo`
  metadata
