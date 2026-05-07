## ADDED Requirements

### Requirement: Fedora discovery runtime configuration
Bootup SHALL allow operator runtime configuration to override Fedora discovery
source settings.

#### Scenario: Fedora discovery config is supplied
- **WHEN** provider runtime configuration includes Fedora discovery URL or
  discovery timeout fields
- **THEN** bootup SHALL validate those fields and pass them to the Fedora
  provider before discovery can run

#### Scenario: Fedora discovery config is invalid
- **WHEN** Fedora discovery URL or discovery timeout configuration is malformed
- **THEN** bootup SHALL fail startup before registering provider targets
