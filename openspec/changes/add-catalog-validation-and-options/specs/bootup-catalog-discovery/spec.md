## ADDED Requirements

### Requirement: Catalog list metadata
Bootup SHALL present compact catalog list output that includes enough metadata
to distinguish static targets.

#### Scenario: Operator lists targets
- **WHEN** an operator lists available targets
- **THEN** bootup SHALL show each target ID, display name, distribution,
  release, architecture, provider, and boot action

### Requirement: Catalog show details
Bootup SHALL present detailed target information for a selected catalog target.

#### Scenario: Operator shows target details
- **WHEN** an operator asks to show one target
- **THEN** bootup SHALL show the target metadata, provider, boot action,
  artifact references, lifecycle decoration, and declared option definitions

#### Scenario: Unknown target is shown
- **WHEN** an operator asks to show a target ID that is not registered
- **THEN** bootup SHALL fail with a clear error and SHALL NOT stage artifacts

### Requirement: Catalog output stability
Bootup SHALL keep catalog discovery output stable enough for operators to scan
and for tests to assert core fields.

#### Scenario: Target has no optional metadata
- **WHEN** a listed or shown target omits optional lifecycle or option metadata
- **THEN** bootup SHALL render the target without placeholder noise or malformed
  output
