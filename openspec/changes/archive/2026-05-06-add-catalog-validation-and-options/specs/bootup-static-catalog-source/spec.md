## ADDED Requirements

### Requirement: Static catalog option source metadata
Bootup SHALL allow structured static catalog source entries to declare target
option definitions.

#### Scenario: Generated target includes options
- **WHEN** a structured catalog source target declares option definitions
- **THEN** `go generate ./internal/catalog` SHALL preserve those definitions in
  the generated embedded catalog

#### Scenario: Catalog option metadata is invalid
- **WHEN** a structured catalog source target declares an option with missing
  ID, unsupported type, invalid allowed values, or malformed command-line
  behavior
- **THEN** catalog generation or catalog loading SHALL reject the target before
  provider registration

### Requirement: Static catalog option data remains non-executable
Bootup SHALL treat catalog option definitions as validated data only.

#### Scenario: Catalog option attempts executable behavior
- **WHEN** a static catalog option definition attempts to load runtime code,
  run scripts, or perform dynamic discovery
- **THEN** bootup SHALL reject or ignore that behavior because static catalog
  options MUST NOT be executable policy
