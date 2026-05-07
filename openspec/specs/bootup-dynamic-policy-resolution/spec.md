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

#### Scenario: Remote policy is fetched over HTTPS
- **WHEN** an operator configures a remote policy URL
- **THEN** bootup SHALL require HTTPS transport and local signature trust
  material before using the policy response

#### Scenario: Remote policy cache fallback is used
- **WHEN** a remote policy fetch fails and a cache file is configured
- **THEN** bootup MAY load the cached policy body only if it passes the same
  signature and freshness checks required for a freshly fetched policy response

#### Scenario: Remote policy cache is unauthenticated or stale
- **WHEN** a cached remote policy body fails authentication, is malformed, is
  expired, or exceeds the configured maximum age
- **THEN** bootup SHALL reject the cached body before provider planning

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

### Requirement: Policy operations
Bootup SHALL provide operator-facing support for producing and exercising signed
policy decisions without weakening the data-only policy contract.

#### Scenario: Operator signs a local policy document
- **WHEN** an operator uses the supported signing flow for a policy JSON
  document
- **THEN** the produced detached signature and public key SHALL be compatible
  with bootup's signed policy runtime flags

#### Scenario: Policy smoke selects a catalog target
- **WHEN** the signed policy smoke runs
- **THEN** it SHALL verify that a signed policy selects an existing catalog
  target, validates the selected options, and can emit redacted diagnostics for
  the policy run

### Requirement: Interactive policy fallback
Bootup SHALL keep policy failure fail-closed by default and SHALL only return to
manual selection when explicitly configured for an interactive run.

#### Scenario: Interactive fallback is explicit
- **WHEN** menu mode is configured to try policy first and the operator enables
  manual fallback
- **THEN** bootup SHALL report the policy failure category and start manual
  target selection instead of staging a policy-selected target

#### Scenario: Non-interactive policy fails
- **WHEN** a non-interactive policy run cannot produce a valid decision
- **THEN** bootup SHALL fail closed before provider planning even if a fallback
  option was not explicitly configured for an interactive run
