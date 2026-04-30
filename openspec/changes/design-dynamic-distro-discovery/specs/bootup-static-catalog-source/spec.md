## MODIFIED Requirements

### Requirement: Hosted and dynamic catalogs are deferred
Bootup SHALL keep runtime URL-hosted catalogs and dynamic distro discovery out
of the implemented static catalog source.

#### Scenario: Catalog URL is not supported
- **WHEN** an operator needs a URL-hosted static catalog
- **THEN** bootup SHALL require a future catalog authenticity and freshness
  design before adding URL loading behavior

#### Scenario: Static catalog does not perform dynamic discovery
- **WHEN** bootup lists targets from a static catalog document
- **THEN** it SHALL NOT discover new distro releases, architectures, install
  options, end-of-life status, or script-driven boot policy from that static
  catalog document at runtime

#### Scenario: Dynamic discovery is a separate provider mode
- **WHEN** bootup implements dynamic distro discovery
- **THEN** it SHALL do so through compiled-in provider discovery behavior rather
  than by extending static catalog documents into executable discovery logic
