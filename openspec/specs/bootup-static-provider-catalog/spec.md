# bootup-static-provider-catalog Specification

## Purpose
Define bootup's static catalog metadata contract for concrete boot targets
exposed by compiled-in providers.
## Requirements
### Requirement: Static provider target catalog
Bootup SHALL expose a typed catalog metadata model for static, concrete boot
targets provided by compiled-in providers.

#### Scenario: Static target carries catalog metadata
- **WHEN** a compiled-in provider exposes a static concrete boot target
- **THEN** the target SHALL include catalog metadata for distribution, release,
  architecture, and target kind

#### Scenario: Provider target metadata is validated
- **WHEN** bootup collects targets from compiled-in providers
- **THEN** it SHALL reject targets with missing IDs, mismatched provider IDs,
  missing display names, or incomplete catalog metadata before rendering them to
  the operator

#### Scenario: Catalog document supplies provider targets
- **WHEN** bootup starts with an embedded or local static catalog document
- **THEN** compiled-in providers SHALL expose their static targets from that
  validated catalog source

#### Scenario: Operator interface uses catalog metadata
- **WHEN** bootup renders static provider targets in an operator interface
- **THEN** it SHALL use the typed catalog metadata for grouping and labels

#### Scenario: Dynamic modes are not required
- **WHEN** bootup lists static provider catalog targets
- **THEN** it SHALL NOT require runtime provider plugins, remote catalog
  discovery, or script-driven boot policy evaluation

### Requirement: Target boot action metadata
Bootup SHALL validate optional boot action metadata on static provider targets.

#### Scenario: Target declares supported action
- **WHEN** a static target declares a supported boot action
- **THEN** bootup SHALL preserve that action in the target metadata passed to
  the selected provider

#### Scenario: Target distribution differs from provider
- **WHEN** a static target is handled by a generic compiled provider
- **THEN** bootup SHALL allow the target catalog distribution to name the
  target family rather than the compiled provider ID

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

