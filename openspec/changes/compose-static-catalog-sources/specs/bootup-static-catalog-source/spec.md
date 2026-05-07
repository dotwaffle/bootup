## MODIFIED Requirements

### Requirement: Static catalog document source
Bootup SHALL source concrete static provider targets from a versioned static
catalog document.

#### Scenario: Embedded catalog is used by default
- **WHEN** bootup starts without a catalog path or hosted catalog URL
- **THEN** bootup SHALL use its embedded static catalog document as the provider
  target source

#### Scenario: Local catalog replaces embedded catalog
- **WHEN** bootup starts with a local catalog path and does not request default
  catalog inclusion
- **THEN** bootup SHALL load that catalog document instead of the embedded
  static catalog document

#### Scenario: Hosted catalog replaces embedded catalog
- **WHEN** bootup starts with an authenticated hosted catalog URL and does not
  request default catalog inclusion
- **THEN** bootup SHALL load that catalog document instead of the embedded
  static catalog document

#### Scenario: Selected catalog composes with embedded catalog
- **WHEN** bootup starts with a local catalog path or authenticated hosted
  catalog URL and explicitly requests default catalog inclusion
- **THEN** bootup SHALL combine the embedded default catalog targets with the
  selected catalog targets before provider registration

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

#### Scenario: Composed catalog target IDs collide
- **WHEN** composed catalog sources contain the same target ID
- **THEN** bootup SHALL reject the composed catalog before registering provider
  targets

#### Scenario: Catalog references unknown provider
- **WHEN** a catalog document references a provider that is not compiled into
  the current bootup binary
- **THEN** bootup SHALL reject the catalog before registering provider targets

#### Scenario: Catalog JSON is malformed
- **WHEN** a local catalog document cannot be parsed as a supported catalog
- **THEN** bootup SHALL fail startup before registering provider targets
