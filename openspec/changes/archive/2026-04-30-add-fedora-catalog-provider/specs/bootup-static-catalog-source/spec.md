## ADDED Requirements

### Requirement: Generated embedded static catalog
Bootup SHALL generate the embedded default static catalog from a structured
repository source file.

#### Scenario: Generated catalog is current
- **WHEN** the structured catalog source changes
- **THEN** `go generate ./internal/catalog` SHALL produce the embedded
  `default.json` deterministically

#### Scenario: Generated catalog metadata is preserved
- **WHEN** a generated catalog source target includes source or lifecycle
  metadata
- **THEN** the generated embedded catalog SHALL preserve that metadata

#### Scenario: Generated catalog is stale
- **WHEN** the embedded generated catalog no longer matches the structured
  source
- **THEN** repository tests SHALL fail before provider registration behavior can
  silently drift

### Requirement: Static lifecycle metadata source
Bootup SHALL allow the structured static catalog source to include
informational lifecycle metadata for generated static targets.

#### Scenario: Static lifecycle metadata is generated
- **WHEN** a catalog source target includes lifecycle status and source fields
- **THEN** bootup SHALL expose that lifecycle decoration on the corresponding
  static target

#### Scenario: Static lifecycle metadata remains informational
- **WHEN** bootup verifies downloaded boot artifacts
- **THEN** it MUST NOT use generated lifecycle metadata as signature, checksum,
  keyring, transport, or trust material

## MODIFIED Requirements

### Requirement: Default static catalog targets
Bootup SHALL include a default static catalog with the initial compiled-in
provider target set.

#### Scenario: Fedora targets are in default catalog
- **WHEN** bootup starts with the default static catalog
- **THEN** it SHALL expose Fedora Server amd64 netboot targets as selectable
  static targets

### Requirement: Hosted and dynamic catalogs are deferred
Bootup SHALL keep runtime URL-hosted catalogs and dynamic distro discovery out
of the implemented static catalog source.

#### Scenario: Hosted catalog trust model is explicit
- **WHEN** a future bootup version adds URL-hosted static catalog loading
- **THEN** it SHALL define catalog authenticity, freshness, cache behavior,
  offline fallback, and operator trust configuration before loading hosted
  catalog content at runtime
