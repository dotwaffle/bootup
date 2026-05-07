## MODIFIED Requirements

### Requirement: Static catalog document source
Bootup SHALL source concrete static provider targets from a versioned static
catalog document.

#### Scenario: Embedded catalog is used by default
- **WHEN** bootup starts without a catalog path or hosted catalog URL
- **THEN** bootup SHALL use its embedded static catalog document as the provider
  target source

#### Scenario: Local catalog replaces embedded catalog
- **WHEN** bootup starts with a local catalog path
- **THEN** bootup SHALL load that catalog document instead of the embedded
  static catalog document

#### Scenario: Hosted catalog replaces embedded catalog
- **WHEN** bootup starts with an authenticated hosted catalog URL
- **THEN** bootup SHALL load that catalog document instead of the embedded
  static catalog document

#### Scenario: Catalog source metadata is loaded
- **WHEN** a static catalog target includes optional source metadata
- **THEN** bootup SHALL pass that source metadata to the compiled-in provider
  selected for the target

#### Scenario: Catalog source is data only
- **WHEN** bootup loads a static catalog document
- **THEN** the document SHALL describe concrete targets for compiled-in
  providers and MUST NOT cause bootup to load provider code from the network or
  from runtime plugin files

### Requirement: Hosted and dynamic catalogs are deferred
Bootup SHALL support authenticated URL-hosted static catalog documents while
keeping dynamic discovery and executable policy out of the static catalog
source.

#### Scenario: Catalog URL requires trust configuration
- **WHEN** an operator needs a URL-hosted static catalog
- **THEN** bootup SHALL require catalog authenticity and freshness configuration
  before adding URL-loaded targets to the provider registry

#### Scenario: Hosted catalog design is documented
- **WHEN** an operator needs a URL-hosted static catalog
- **THEN** bootup SHALL document catalog authenticity, freshness, cache
  behavior, offline fallback, and operator trust configuration for runtime URL
  catalog loading

#### Scenario: Hosted catalog trust model is explicit
- **WHEN** bootup loads URL-hosted static catalog content at runtime
- **THEN** it SHALL define and enforce catalog authenticity, freshness, cache
  behavior, offline fallback, and operator trust configuration before loading
  that hosted catalog content

#### Scenario: Static catalog does not perform dynamic discovery
- **WHEN** bootup lists targets from a static catalog document
- **THEN** it SHALL NOT discover new distro releases, architectures, install
  options, end-of-life status, or script-driven boot policy from that static
  catalog document at runtime

#### Scenario: Dynamic discovery is a separate provider mode
- **WHEN** bootup implements dynamic distro discovery
- **THEN** it SHALL do so through compiled-in provider discovery behavior rather
  than by extending static catalog documents into executable discovery logic

## ADDED Requirements

### Requirement: Static catalog document freshness metadata
Bootup SHALL allow static catalog documents to carry optional publication and
expiry metadata for hosted catalog freshness validation.

#### Scenario: Catalog document includes freshness metadata
- **WHEN** a static catalog document includes publication or expiry timestamps
- **THEN** bootup SHALL parse those timestamps as catalog document metadata and
  preserve normal target validation behavior

#### Scenario: Catalog document freshness metadata is malformed
- **WHEN** a static catalog document includes malformed publication or expiry
  timestamps
- **THEN** bootup SHALL reject the catalog before registering provider targets
