## MODIFIED Requirements

### Requirement: Operator provider runtime configuration
Bootup SHALL allow operators to supply provider source, discovery, lifecycle,
and verification inputs for compiled-in providers through an explicit runtime
configuration file.

#### Scenario: Fedora provider config is supplied
- **WHEN** provider runtime configuration includes Fedora release URL or
  kernel/initrd hash pins
- **THEN** bootup SHALL validate those fields and pass them to the Fedora
  provider before target planning or artifact staging

#### Scenario: Fedora provider config is invalid
- **WHEN** Fedora release URL or hash pin configuration is malformed
- **THEN** bootup SHALL fail startup before registering provider targets
