# bootup-dynamic-policy-resolution Specification

## Purpose
TBD - created by archiving change design-dynamic-policy-resolution. Update Purpose after archive.
## Requirements
### Requirement: Data-only policy decisions
Bootup SHALL treat dynamic policy as authenticated data that selects already
known targets and validated inputs, not as executable behavior.

#### Scenario: Policy selects an existing target
- **WHEN** a policy decision references a target ID already present in the
  current bootup target inventory
- **THEN** bootup SHALL validate that target through the normal provider
  planning path before staging

#### Scenario: Policy attempts executable behavior
- **WHEN** a policy source attempts to run scripts, load plugins, define
  providers, create targets, override artifact trust, or inject arbitrary boot
  arguments
- **THEN** bootup SHALL reject the policy decision before planning, staging, or
  handoff

### Requirement: Policy trust and freshness
Bootup SHALL authenticate and freshness-check dynamic policy decisions before
using them.

#### Scenario: Policy trust material is absent
- **WHEN** an operator enables dynamic policy without required local trust
  material for the configured policy source
- **THEN** bootup SHALL fail before resolving a target decision

#### Scenario: Policy decision is authenticated and fresh
- **WHEN** a policy decision passes every configured authenticity and freshness
  check
- **THEN** bootup MAY use the decision for target and option validation

#### Scenario: Policy decision is unavailable or stale
- **WHEN** a policy source times out, cannot be fetched, fails authentication,
  is malformed, is expired, or exceeds the configured maximum age
- **THEN** bootup SHALL fail closed before provider planning unless an
  interactive operator explicitly falls back to manual target selection

### Requirement: Policy result validation
Bootup SHALL validate policy-selected targets, non-secret options, and secret
references against the current target inventory before provider planning.

#### Scenario: Policy selects non-secret options
- **WHEN** a policy decision selects option values for the chosen target
- **THEN** bootup SHALL validate those options with the same target option
  validation used for operator-selected values

#### Scenario: Policy references secrets
- **WHEN** a policy decision references target-declared secret IDs
- **THEN** bootup SHALL resolve those references through the secret input
  delivery capability and SHALL NOT accept inline secret values from policy
  output

#### Scenario: Policy output is unsupported
- **WHEN** a policy decision references an unknown target, unsupported option,
  invalid option value, undeclared secret ID, missing required secret, or
  unsupported boot action
- **THEN** bootup SHALL reject the decision before staging artifacts

### Requirement: Policy diagnostics
Bootup SHALL report policy posture without exposing sensitive data.

#### Scenario: Policy diagnostics are written
- **WHEN** diagnostics are enabled for a run that evaluates dynamic policy
- **THEN** bootup MAY record policy source posture, decision ID, target ID,
  selected option IDs, secret reference IDs, freshness timestamps, and failure
  categories, and MUST NOT record policy response bodies, secret values, secret
  paths, trust private material, provider config contents, or unredacted
  sensitive option values
