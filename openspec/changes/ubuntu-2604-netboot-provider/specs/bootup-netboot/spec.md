## MODIFIED Requirements

### Requirement: Build-time provider modules
Bootup SHALL support operating system providers compiled into the distributed
image at build time.

#### Scenario: Ubuntu provider is available in the image
- **WHEN** bootup starts with the default provider set
- **THEN** bootup SHALL list Ubuntu 26.04 amd64 netboot as a selectable target

### Requirement: Verified artifact chain
Bootup SHALL verify downloaded target boot artifacts before staging them for
kexec.

#### Scenario: Ubuntu netboot hashes are absent
- **WHEN** the selected Ubuntu provider lacks explicit netboot kernel and
  initrd hashes
- **THEN** bootup SHALL stage Ubuntu netboot artifacts only from HTTPS URLs

#### Scenario: Ubuntu netboot hashes are present
- **WHEN** the Ubuntu provider has release signing trust material and explicit
  netboot kernel and initrd hashes
- **THEN** bootup SHALL verify the signed release checksum file and each
  downloaded netboot artifact before staging it
