## MODIFIED Requirements

### Requirement: Operator provider runtime configuration
Bootup SHALL load operator-supplied provider runtime configuration before
provider registration and pass validated config to compiled-in providers.

#### Scenario: Provider discovery config is supplied
- **WHEN** provider runtime configuration includes discovery URL and discovery
  timeout fields for a compiled-in provider
- **THEN** bootup SHALL validate those fields and pass them to that provider
  before discovery can run

#### Scenario: Provider lifecycle config is supplied
- **WHEN** provider runtime configuration includes lifecycle metadata for a
  provider release
- **THEN** bootup SHALL validate lifecycle status, source, and date fields
  before provider registration

#### Scenario: Provider discovery config is invalid
- **WHEN** discovery URL, discovery timeout, lifecycle status, or lifecycle date
  configuration is malformed
- **THEN** bootup SHALL fail startup before registering provider targets
