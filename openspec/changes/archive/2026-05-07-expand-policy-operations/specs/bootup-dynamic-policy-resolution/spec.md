## MODIFIED Requirements

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

## ADDED Requirements

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
