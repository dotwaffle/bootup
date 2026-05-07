## MODIFIED Requirements

### Requirement: Fedora discovery runtime configuration
Bootup SHALL allow operator runtime configuration to override Fedora discovery
source settings and attach informational lifecycle decoration.

#### Scenario: Fedora discovery config is supplied
- **WHEN** provider runtime configuration includes Fedora discovery URL, local
  discovery metadata path, discovery timeout fields, or lifecycle metadata
- **THEN** bootup SHALL validate those fields and pass them to the Fedora
  provider before discovery can run

#### Scenario: Fedora discovery lifecycle config is supplied
- **WHEN** provider runtime configuration includes Fedora lifecycle metadata for
  a release
- **THEN** bootup SHALL validate lifecycle status, source, and date fields and
  pass the entry to the Fedora provider before discovery can run

#### Scenario: Fedora discovery config is invalid
- **WHEN** Fedora discovery URL, local discovery metadata path, discovery
  timeout, or lifecycle configuration is malformed
- **THEN** bootup SHALL fail startup before registering provider targets
