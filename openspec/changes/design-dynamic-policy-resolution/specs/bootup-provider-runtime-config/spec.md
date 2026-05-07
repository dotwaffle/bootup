## MODIFIED Requirements

### Requirement: Runtime configuration dynamic policy boundary
Bootup SHALL treat provider runtime configuration as declarative data for
compiled-in providers, not as executable or remote dynamic policy.

#### Scenario: Runtime config attempts policy execution
- **WHEN** provider runtime configuration attempts to provide scripts, runtime
  plugins, policy service decisions, or other unsupported policy fields
- **THEN** bootup SHALL reject the configuration before provider registration,
  target discovery, planning, staging, or handoff

#### Scenario: Dynamic policy uses separate configuration
- **WHEN** an operator needs dynamic policy target selection
- **THEN** bootup SHALL configure and evaluate that policy through a dedicated
  policy resolver capability rather than provider runtime configuration fields

#### Scenario: Declarative provider config remains supported
- **WHEN** provider runtime configuration contains only supported typed provider
  source, discovery, lifecycle, and verification fields
- **THEN** bootup SHALL continue to apply that data to compiled-in providers
