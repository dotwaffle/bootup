## ADDED Requirements

### Requirement: Generated static artifact hash pins
Bootup SHALL preserve validated source artifact hash pins when generating the
embedded static catalog from the structured source document.

#### Scenario: Generated catalog includes source artifact pins
- **WHEN** a structured catalog source target includes kernel or initrd
  SHA-256 source hashes
- **THEN** `go generate ./internal/catalog` SHALL preserve those hashes in the
  generated embedded catalog

#### Scenario: Generated catalog source has malformed artifact pins
- **WHEN** a structured catalog source target includes malformed or partial
  kernel/initrd source hashes
- **THEN** catalog generation SHALL reject the source target before writing
  generated catalog output
