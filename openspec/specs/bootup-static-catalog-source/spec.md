# bootup-static-catalog-source Specification

## Purpose
Define bootup's static catalog document sources for concrete provider targets,
including the embedded default catalog and local replacement catalogs.
## Requirements
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

#### Scenario: Existing default targets remain in default catalog
- **WHEN** bootup starts with the default static catalog
- **THEN** it SHALL continue to expose Debian trixie amd64 netboot and Ubuntu
  26.04 amd64 netboot as selectable static targets

#### Scenario: Debian forky target is in default catalog
- **WHEN** bootup starts with the default static catalog
- **THEN** it SHALL expose Debian forky amd64 netboot as a selectable static
  target

#### Scenario: Ubuntu point release targets are in default catalog
- **WHEN** bootup starts with the default static catalog
- **THEN** it SHALL expose Ubuntu 24.04.4 amd64 netboot, Ubuntu 25.10 amd64
  netboot, and Ubuntu 26.04 amd64 netboot as selectable static targets

#### Scenario: Fedora targets are in default catalog
- **WHEN** bootup starts with the default static catalog
- **THEN** it SHALL expose Fedora Server amd64 netboot targets as selectable
  static targets

### Requirement: Hosted and dynamic catalogs are deferred
Bootup SHALL keep runtime URL-hosted catalogs and dynamic distro discovery out
of the implemented static catalog source.

#### Scenario: Catalog URL is not supported
- **WHEN** an operator needs a URL-hosted static catalog
- **THEN** bootup SHALL require a future catalog authenticity and freshness
  design before adding URL loading behavior

#### Scenario: Hosted catalog design is documented
- **WHEN** an operator needs a URL-hosted static catalog
- **THEN** bootup SHALL document that catalog authenticity, freshness, cache
  behavior, offline fallback, and operator trust configuration must be designed
  before runtime URL catalog loading is implemented

#### Scenario: Hosted catalog trust model is explicit
- **WHEN** a future bootup version adds URL-hosted static catalog loading
- **THEN** it SHALL define catalog authenticity, freshness, cache behavior,
  offline fallback, and operator trust configuration before loading hosted
  catalog content at runtime

#### Scenario: Static catalog does not perform dynamic discovery
- **WHEN** bootup lists targets from a static catalog document
- **THEN** it SHALL NOT discover new distro releases, architectures, install
  options, end-of-life status, or script-driven boot policy from that static
  catalog document at runtime

#### Scenario: Dynamic discovery is a separate provider mode
- **WHEN** bootup implements dynamic distro discovery
- **THEN** it SHALL do so through compiled-in provider discovery behavior rather
  than by extending static catalog documents into executable discovery logic

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

### Requirement: Static Linux source metadata
Bootup SHALL allow static catalog targets for the generic Linux provider to
describe kernel path, optional initrd path, and command line source metadata.

#### Scenario: Generic Linux source target is loaded
- **WHEN** a catalog target references the generic Linux provider
- **THEN** bootup SHALL validate source base URL, kernel path, optional initrd
  path, and command line metadata before registering the target

### Requirement: Extended default utility targets
Bootup SHALL include Linux-shaped utility and installer targets in the embedded
default catalog.

#### Scenario: Local boot target is in default catalog
- **WHEN** bootup starts with the default static catalog
- **THEN** it SHALL expose a local disk boot target

#### Scenario: openSUSE target is in default catalog
- **WHEN** bootup starts with the default static catalog
- **THEN** it SHALL expose an openSUSE Leap amd64 installer target

#### Scenario: Arch Linux target is in default catalog
- **WHEN** bootup starts with the default static catalog
- **THEN** it SHALL expose an Arch Linux amd64 netboot target

#### Scenario: GParted target is in default catalog
- **WHEN** bootup starts with the default static catalog
- **THEN** it SHALL expose a GParted Live amd64 target

#### Scenario: Non-kexec diagnostic target is excluded
- **WHEN** bootup starts with the default static catalog
- **THEN** it SHALL NOT expose diagnostic utility targets that require
  firmware, memdisk, Multiboot, COM32, or bootloader-specific handoff semantics
  unless a compatible boot action exists
