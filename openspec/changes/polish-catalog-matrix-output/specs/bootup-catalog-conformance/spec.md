## MODIFIED Requirements

### Requirement: Catalog conformance matrix
Bootup SHALL provide a non-interactive catalog conformance matrix for the
currently configured provider registry.

#### Scenario: Operator renders catalog matrix
- **WHEN** an operator selects the catalog matrix mode
- **THEN** bootup SHALL render each registered target with target ID, provider,
  distribution, release, architecture, kind, lifecycle status, resolved boot
  action, plan status, artifact trust classification, and smoke coverage
  classification

#### Scenario: Catalog matrix is hermetic
- **WHEN** bootup renders the catalog matrix
- **THEN** it SHALL request offline provider planning and SHALL NOT download
  boot artifacts, stage artifacts, contact upstream mirrors, fetch remote
  metadata, or launch QEMU

#### Scenario: Target plan succeeds
- **WHEN** a registered target can be planned by its provider
- **THEN** the matrix SHALL report that target with a successful plan status

#### Scenario: Target plan fails
- **WHEN** a registered target cannot be planned by its provider
- **THEN** the matrix SHALL include the target and report the planning error
  without hiding other targets
