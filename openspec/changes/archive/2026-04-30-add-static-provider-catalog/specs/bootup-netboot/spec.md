## MODIFIED Requirements

### Requirement: Build-time provider modules
Bootup SHALL support operating system providers compiled into the distributed
image at build time.

#### Scenario: Provider is available in the image
- **WHEN** bootup starts with a provider included at build time
- **THEN** bootup SHALL list that provider's supported boot targets through the
  operator interface

#### Scenario: Ubuntu provider is available in the image
- **WHEN** bootup starts with the default provider set
- **THEN** bootup SHALL list Ubuntu 26.04 amd64 netboot as a selectable target

#### Scenario: Catalog metadata is available
- **WHEN** a provider exposes static concrete boot targets
- **THEN** each target SHALL include typed catalog metadata that allows future
  UIs to group entries without loading runtime plugins

#### Scenario: Runtime provider loading is absent
- **WHEN** bootup is running in the target environment
- **THEN** bootup MUST NOT require loading provider code from the network or
  from runtime plugin files
