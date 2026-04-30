## MODIFIED Requirements

### Requirement: Default static catalog targets
Bootup SHALL include a default static catalog with the initial compiled-in
provider target set.

#### Scenario: Debian forky target is in default catalog
- **WHEN** bootup starts with the default static catalog
- **THEN** it SHALL expose Debian forky amd64 netboot as a selectable static
  target

### Requirement: Hosted and dynamic catalogs are deferred
Bootup SHALL keep runtime URL-hosted catalogs and dynamic distro discovery out
of the implemented static catalog source.

#### Scenario: Hosted catalog design is documented
- **WHEN** an operator needs a URL-hosted static catalog
- **THEN** bootup SHALL document that catalog authenticity, freshness, cache
  behavior, offline fallback, and operator trust configuration must be designed
  before runtime URL catalog loading is implemented
