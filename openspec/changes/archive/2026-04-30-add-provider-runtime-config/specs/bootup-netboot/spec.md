## MODIFIED Requirements

### Requirement: Verified artifact chain
Bootup SHALL verify downloaded target boot artifacts before staging them for
kexec when provider verification material is available, and SHALL otherwise
constrain explicitly documented HTTPS-only provider paths to HTTPS URLs.

#### Scenario: Runtime config supplies provider verification material
- **WHEN** bootup starts with provider runtime configuration containing
  verification material for a compiled-in provider
- **THEN** bootup SHALL pass that verification material to provider boot
  planning and staging before artifact retrieval

#### Scenario: Debian metadata verifies successfully
- **WHEN** the Debian provider downloads archive metadata and installer
  checksum data
- **THEN** bootup SHALL validate the signed metadata against explicitly
  configured Debian archive trust material before trusting installer checksums

#### Scenario: Debian trust material is absent
- **WHEN** the selected Debian provider has no configured archive trust
  material
- **THEN** bootup SHALL fail closed before staging artifacts and MUST NOT
  execute kexec

#### Scenario: Artifact checksum matches trusted metadata
- **WHEN** bootup downloads the selected Debian Installer kernel and initrd
- **THEN** bootup SHALL verify each artifact against trusted checksum metadata
  before staging it

#### Scenario: Verification fails
- **WHEN** signature validation or artifact checksum validation fails
- **THEN** bootup SHALL fail closed, report the verification error, and MUST NOT
  execute kexec

#### Scenario: Ubuntu netboot hashes are absent
- **WHEN** the selected Ubuntu provider lacks explicit netboot kernel and
  initrd hashes
- **THEN** bootup SHALL stage Ubuntu netboot artifacts only from HTTPS URLs

#### Scenario: Ubuntu netboot hashes are present
- **WHEN** the Ubuntu provider has release signing trust material and explicit
  netboot kernel and initrd hashes
- **THEN** bootup SHALL verify the signed release checksum file and each
  downloaded netboot artifact before staging it
