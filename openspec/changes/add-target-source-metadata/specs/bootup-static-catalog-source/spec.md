## MODIFIED Requirements

### Requirement: Static catalog document source
Bootup SHALL source concrete static provider targets from a versioned static
catalog document.

#### Scenario: Embedded catalog is used by default
- **WHEN** bootup starts without a catalog path
- **THEN** bootup SHALL use its embedded static catalog document as the provider
  target source

#### Scenario: Local catalog replaces embedded catalog
- **WHEN** bootup starts with a local catalog path
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

### Requirement: Static catalog validation
Bootup SHALL validate static catalog documents before provider target discovery.

#### Scenario: Catalog schema is unsupported
- **WHEN** a catalog document uses an unsupported schema version
- **THEN** bootup SHALL fail startup before registering provider targets

#### Scenario: Catalog target metadata is incomplete
- **WHEN** a catalog target is missing an ID, provider ID, display name,
  distribution, release, architecture, or target kind
- **THEN** bootup SHALL reject the catalog before registering provider targets

#### Scenario: Catalog target source metadata is invalid
- **WHEN** a catalog target includes malformed source metadata
- **THEN** bootup SHALL reject the catalog before registering provider targets

#### Scenario: Catalog target IDs collide
- **WHEN** a catalog document contains duplicate target IDs
- **THEN** bootup SHALL reject the catalog before registering provider targets

#### Scenario: Catalog references unknown provider
- **WHEN** a catalog document references a provider that is not compiled into
  the current bootup binary
- **THEN** bootup SHALL reject the catalog before registering provider targets

#### Scenario: Catalog JSON is malformed
- **WHEN** a local catalog document cannot be parsed as a supported catalog
- **THEN** bootup SHALL fail startup before registering provider targets

### Requirement: Default static catalog targets
Bootup SHALL include a default static catalog with the initial compiled-in
provider target set.

#### Scenario: Debian bullseye target is in default catalog
- **WHEN** bootup starts with the default static catalog
- **THEN** it SHALL expose Debian bullseye amd64 netboot as a selectable static
  target

#### Scenario: Debian bookworm target is in default catalog
- **WHEN** bootup starts with the default static catalog
- **THEN** it SHALL expose Debian bookworm amd64 netboot as a selectable static
  target

#### Scenario: Ubuntu point release targets are in default catalog
- **WHEN** bootup starts with the default static catalog
- **THEN** it SHALL expose Ubuntu 24.04.4 amd64 netboot, Ubuntu 25.10 amd64
  netboot, and Ubuntu 26.04 amd64 netboot as selectable static targets

#### Scenario: Existing default targets remain in default catalog
- **WHEN** bootup starts with the default static catalog
- **THEN** it SHALL continue to expose Debian trixie amd64 netboot and Ubuntu
  26.04 amd64 netboot as selectable static targets

### Requirement: Hosted and dynamic catalogs are deferred
Bootup SHALL keep runtime URL-hosted catalogs and dynamic distro discovery out
of the implemented static catalog source.

#### Scenario: Catalog URL is not supported
- **WHEN** an operator needs a URL-hosted static catalog
- **THEN** bootup SHALL require a future catalog authenticity and freshness
  design before adding URL loading behavior

#### Scenario: Dynamic discovery is not performed
- **WHEN** bootup lists targets from a static catalog document
- **THEN** it SHALL NOT discover new distro releases, architectures, install
  options, end-of-life status, or script-driven boot policy at runtime
