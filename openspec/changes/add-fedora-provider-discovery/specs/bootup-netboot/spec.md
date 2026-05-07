## ADDED Requirements

### Requirement: Fedora provider discovery family
Bootup SHALL include Fedora in the default provider set's runtime discovery
families.

#### Scenario: Fedora discovery family is registered
- **WHEN** bootup starts with the default provider set
- **THEN** the provider registry SHALL expose a Fedora discovery family in
  addition to static Fedora catalog targets
