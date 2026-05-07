## MODIFIED Requirements

### Requirement: Target option secret boundary
Bootup SHALL treat catalog target options as non-secret boot argument data
unless a separate secret-safe delivery capability explicitly handles the secret
input outside option command-line expansion.

#### Scenario: Secret target option is rejected
- **WHEN** a static catalog target declares an option with a secret marker
- **THEN** bootup SHALL reject that target before rendering, planning, staging,
  or handoff

#### Scenario: Non-secret option output remains inspectable
- **WHEN** an operator selects a valid non-secret target option
- **THEN** bootup SHALL continue to render the resulting boot command-line or
  boot action argument data in diagnostics

#### Scenario: Secret delivery uses a separate capability
- **WHEN** a target needs a password, password hash, SSH key, token, or other
  secret input
- **THEN** that input MUST use a secret delivery declaration and MUST NOT be
  represented as a current target option command-line or loader-argument
  fragment
