## ADDED Requirements

### Requirement: Static target option metadata
Bootup SHALL expose validated option metadata on static provider targets.

#### Scenario: Provider target has options
- **WHEN** bootup collects a static provider target with catalog-declared
  options
- **THEN** the target metadata SHALL preserve those options for operator
  discovery and provider planning

#### Scenario: Provider target option IDs collide
- **WHEN** a static provider target declares duplicate option IDs
- **THEN** bootup SHALL reject the target before rendering it to the operator

### Requirement: Static target option planning input
Bootup SHALL pass selected target options to provider planning through explicit
planning input.

#### Scenario: Operator selects target options
- **WHEN** an operator selects valid options for a static target
- **THEN** the selected options SHALL be available to the provider plan step
  without requiring providers to inspect CLI state or global variables
