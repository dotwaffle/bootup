## ADDED Requirements

### Requirement: Dynamic distro discovery families
Bootup SHALL allow compiled-in providers to expose dynamic discovery families
that can discover concrete boot targets at runtime.

#### Scenario: Provider family is listed
- **WHEN** a compiled-in provider supports dynamic distro discovery
- **THEN** bootup SHALL expose a provider family entry that the operator can
  select before concrete target discovery

#### Scenario: Discovery is explicit
- **WHEN** bootup lists static concrete targets
- **THEN** it SHALL NOT run dynamic distro discovery until the operator selects
  a provider family or a non-interactive discovery mode requests it

#### Scenario: Provider code remains compiled in
- **WHEN** bootup runs dynamic distro discovery
- **THEN** it MUST use provider logic already compiled into the bootup binary
  and MUST NOT load provider code from remote catalogs or runtime plugins

### Requirement: Discovered concrete targets
Dynamic distro discovery SHALL return concrete provider targets compatible with
boot planning and staging.

#### Scenario: Discovery returns releases and architectures
- **WHEN** dynamic discovery succeeds for a provider family
- **THEN** bootup SHALL receive concrete targets with distribution, release,
  architecture, kind, display name, and provider source metadata as needed

#### Scenario: Discovered target is selected
- **WHEN** the operator selects a discovered concrete target
- **THEN** bootup SHALL plan, verify, stage, and hand off that target through
  the same provider planning path used for static catalog targets

#### Scenario: Discovery finds no targets
- **WHEN** dynamic discovery succeeds but no supported concrete targets are
  available
- **THEN** bootup SHALL report that result clearly and MUST NOT fabricate static
  fallback targets

### Requirement: Discovery failure handling
Bootup SHALL report dynamic discovery failures without corrupting the static
catalog target list.

#### Scenario: Discovery source fails
- **WHEN** a provider discovery source is unavailable, malformed, or times out
- **THEN** bootup SHALL report the discovery failure to the operator and keep
  the current stage-1 environment available for diagnosis

#### Scenario: Static catalog remains available
- **WHEN** dynamic discovery fails for one provider family
- **THEN** bootup SHALL preserve already-loaded static catalog targets and other
  provider families

### Requirement: Lifecycle decoration
Bootup SHALL allow dynamic discovery results to include optional lifecycle
decoration without treating lifecycle metadata as verification material.

#### Scenario: Lifecycle status is available
- **WHEN** a provider attaches lifecycle metadata to a discovered target
- **THEN** bootup SHALL expose that metadata as informational target decoration

#### Scenario: Lifecycle status is absent
- **WHEN** lifecycle metadata is unavailable for a discovered target
- **THEN** bootup SHALL still allow the target to be selected if provider
  planning and verification requirements can be satisfied

#### Scenario: Lifecycle status is not a trust signal
- **WHEN** bootup verifies downloaded target boot artifacts
- **THEN** it MUST NOT treat lifecycle metadata as signature, checksum, keyring,
  transport, or trust material
