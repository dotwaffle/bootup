## ADDED Requirements

### Requirement: Provider discovery mode
Bootup SHALL support a future provider discovery mode that complements static
target listing without requiring runtime provider plugins.

#### Scenario: Discovery-capable provider is compiled in
- **WHEN** bootup starts with a provider that supports dynamic distro discovery
- **THEN** bootup SHALL be able to expose that provider's discovery family
  alongside static concrete targets

#### Scenario: Discovered target uses normal boot path
- **WHEN** the operator selects a discovered concrete target
- **THEN** bootup SHALL use the same provider boot planning, artifact
  verification, staging, and kexec handoff behavior used for static targets

#### Scenario: Runtime provider loading remains absent
- **WHEN** bootup performs provider discovery
- **THEN** bootup MUST NOT require loading provider code from the network or
  from runtime plugin files
