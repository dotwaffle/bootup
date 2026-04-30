## MODIFIED Requirements

### Requirement: Provider discovery mode
Bootup SHALL support provider discovery mode that complements static target
listing without requiring runtime provider plugins.

#### Scenario: Discovery-capable provider is compiled in
- **WHEN** bootup starts with a provider that supports dynamic distro discovery
- **THEN** bootup SHALL expose that provider's discovery family alongside
  static concrete targets
