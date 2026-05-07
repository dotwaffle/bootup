# bootup-hosted-static-catalogs Specification

## Purpose
Define authenticated URL-hosted static catalog loading, including explicit
operator selection, catalog byte authentication, freshness checks, cache
fallback, and the boundary between catalog trust and provider artifact trust.
## Requirements
### Requirement: Hosted catalog URL loading
Bootup SHALL load URL-hosted static catalog documents only when the operator
explicitly selects a hosted catalog source.

#### Scenario: Hosted catalog URL is selected
- **WHEN** bootup starts with a hosted catalog URL
- **THEN** bootup SHALL fetch the catalog document from that URL instead of
  using the embedded static catalog document

#### Scenario: Hosted catalog uses unsupported URL scheme
- **WHEN** bootup is asked to fetch a hosted catalog from an unsupported URL
  scheme
- **THEN** bootup SHALL fail startup before registering provider targets

#### Scenario: Hosted catalog fetch fails
- **WHEN** the hosted catalog source is unavailable, times out, or returns a
  non-success HTTP status
- **THEN** bootup SHALL fail startup unless an authenticated cache fallback is
  explicitly enabled and usable

### Requirement: Hosted catalog authentication
Bootup SHALL authenticate hosted catalog bytes before parsing them as a static
catalog document.

#### Scenario: Digest-pinned hosted catalog matches
- **WHEN** an operator supplies a hosted catalog URL and matching SHA-256 digest
- **THEN** bootup SHALL accept the downloaded bytes for catalog parsing

#### Scenario: Digest-pinned hosted catalog mismatches
- **WHEN** an operator supplies a hosted catalog URL and the downloaded bytes do
  not match the configured SHA-256 digest
- **THEN** bootup SHALL reject the hosted catalog before parsing it

#### Scenario: Signature-authenticated hosted catalog matches
- **WHEN** an operator supplies a hosted catalog URL, detached Ed25519
  signature, and Ed25519 public key
- **THEN** bootup SHALL verify the detached signature over the downloaded bytes
  before parsing the catalog

#### Scenario: Hosted catalog has no trust configuration
- **WHEN** an operator supplies a hosted catalog URL without a digest pin or
  signature trust configuration
- **THEN** bootup SHALL fail startup before registering provider targets

#### Scenario: Multiple trust checks are supplied
- **WHEN** an operator supplies both a SHA-256 digest pin and signature trust
  configuration for a hosted catalog
- **THEN** bootup SHALL require every configured trust check to pass before
  parsing the catalog

### Requirement: Hosted catalog freshness
Bootup SHALL enforce freshness controls for hosted catalog documents before
provider registration.

#### Scenario: Hosted catalog has not expired
- **WHEN** a hosted catalog includes freshness metadata that is acceptable under
  the operator's policy
- **THEN** bootup SHALL allow the catalog to proceed to static catalog
  validation

#### Scenario: Hosted catalog is expired
- **WHEN** a hosted catalog includes an expiry time earlier than the current
  system time
- **THEN** bootup SHALL reject the catalog before registering provider targets

#### Scenario: Hosted catalog exceeds maximum age
- **WHEN** a hosted catalog publication time is older than the configured
  maximum age
- **THEN** bootup SHALL reject the catalog before registering provider targets

#### Scenario: Hosted catalog lacks required freshness metadata
- **WHEN** an operator requires hosted catalog freshness metadata and the
  catalog document omits it
- **THEN** bootup SHALL reject the catalog before registering provider targets

### Requirement: Hosted catalog cache fallback
Bootup SHALL use hosted catalog cache fallback only after authenticating and
validating cached catalog bytes.

#### Scenario: Hosted catalog cache is updated
- **WHEN** bootup successfully fetches, authenticates, freshness-checks, parses,
  and validates a hosted catalog
- **THEN** bootup SHALL update the configured cache path with the authenticated
  catalog bytes

#### Scenario: Hosted catalog cache fallback is used
- **WHEN** hosted catalog fetching fails and cache fallback is enabled
- **THEN** bootup SHALL load the configured cache only if the cached bytes pass
  the same authentication, freshness, and catalog validation checks

#### Scenario: Hosted catalog cache fallback is not enabled
- **WHEN** hosted catalog fetching fails and cache fallback is not enabled
- **THEN** bootup SHALL fail startup before registering provider targets

#### Scenario: Hosted catalog cache is stale or unauthenticated
- **WHEN** cached catalog bytes fail freshness or authentication checks
- **THEN** bootup SHALL reject the cache and fail startup

### Requirement: Hosted catalogs remain static data
Bootup SHALL treat hosted catalogs as static data for compiled-in providers
only.

#### Scenario: Hosted catalog attempts executable behavior
- **WHEN** a hosted catalog attempts to load runtime code, run scripts, define
  provider plugins, or perform dynamic discovery
- **THEN** bootup SHALL reject or ignore that behavior because hosted static
  catalogs MUST NOT be executable policy

#### Scenario: Hosted catalog artifact trust remains provider-owned
- **WHEN** bootup selects a target from a hosted catalog
- **THEN** the selected provider SHALL still perform its normal boot-artifact
  verification using provider-owned trust material and catalog data SHALL NOT
  replace that verification contract
