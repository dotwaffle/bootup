## MODIFIED Requirements

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
- **WHEN** bootup starts with an embedded, local, or authenticated hosted static
  catalog document
- **THEN** compiled-in providers SHALL expose their static targets from that
  validated catalog source

#### Scenario: Operator interface uses catalog metadata
- **WHEN** bootup renders static provider targets in an operator interface
- **THEN** it SHALL use the typed catalog metadata for grouping and labels

#### Scenario: Dynamic modes are not required
- **WHEN** bootup lists static provider catalog targets
- **THEN** it SHALL NOT require runtime provider plugins, remote provider code
  discovery, or script-driven boot policy evaluation
