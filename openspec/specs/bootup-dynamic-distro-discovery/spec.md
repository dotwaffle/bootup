# bootup-dynamic-distro-discovery Specification

## Purpose
Define runtime discovery of concrete boot targets by compiled-in provider
logic, including provider family selection, discovered target compatibility,
failure handling, and informational lifecycle decoration.
## Requirements
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

#### Scenario: Discovery finds no targets
- **WHEN** dynamic discovery succeeds but returns no concrete targets
- **THEN** bootup SHALL report the empty result clearly and keep the current
  stage-1 environment available for diagnosis

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

### Requirement: Ubuntu dynamic discovery
Bootup SHALL allow the compiled-in Ubuntu provider to discover amd64 netboot
targets from a configured Ubuntu release index.

#### Scenario: Ubuntu family is listed
- **WHEN** the Ubuntu provider is compiled into bootup
- **THEN** bootup SHALL expose an Ubuntu discovery family without running
  discovery

#### Scenario: Ubuntu release is discovered
- **WHEN** Ubuntu discovery finds a release with a live-server amd64 ISO and
  amd64 netboot kernel/initrd paths
- **THEN** bootup SHALL return a concrete Ubuntu target with source base URL,
  ISO filename, release, architecture, and target kind metadata

#### Scenario: Ubuntu discovered target plans normally
- **WHEN** the operator selects an Ubuntu discovered target
- **THEN** the Ubuntu provider SHALL plan kernel, initrd, checksum, signature,
  and ISO URLs through the normal provider planning path

### Requirement: Configured lifecycle decoration
Bootup SHALL allow operators to provide provider lifecycle decoration for
discovered targets.

#### Scenario: Lifecycle map matches discovered release
- **WHEN** a provider discovers a target whose release appears in configured
  lifecycle metadata
- **THEN** bootup SHALL attach that lifecycle metadata to the discovered target

#### Scenario: Lifecycle map is absent
- **WHEN** provider lifecycle metadata is not configured for a discovered target
- **THEN** bootup SHALL still allow the target to be selected if provider
  planning and verification requirements can be satisfied

#### Scenario: Lifecycle remains informational
- **WHEN** bootup verifies downloaded boot artifacts
- **THEN** it MUST NOT use lifecycle metadata as signature, checksum, keyring,
  transport, or trust material

### Requirement: Shared provider discovery HTTP helpers
Bootup SHALL keep common provider discovery HTTP request behavior in shared,
tested helper code.

#### Scenario: Provider probes optional discovery artifacts
- **WHEN** a provider probes a candidate discovery URL
- **THEN** shared helper behavior SHALL treat HTTP 404 as absence, report other
  unexpected statuses, and support GET fallback when HEAD is not allowed

#### Scenario: Provider fetches discovery metadata
- **WHEN** a provider fetches discovery metadata through the shared helper
- **THEN** the helper SHALL bind the request to the caller context and return
  response status separately from the response body

### Requirement: Fedora dynamic discovery
Bootup SHALL allow the compiled-in Fedora provider to discover amd64 Server
netboot targets from a configured Fedora releases index.

#### Scenario: Fedora family is listed
- **WHEN** the Fedora provider is compiled into bootup
- **THEN** bootup SHALL expose a Fedora discovery family without running
  discovery

#### Scenario: Fedora release is discovered
- **WHEN** Fedora discovery finds a numeric release with Server x86_64 netboot
  kernel and initrd paths
- **THEN** bootup SHALL return a concrete Fedora target with source base URL,
  release, architecture, and target kind metadata

#### Scenario: Fedora discovered target plans normally
- **WHEN** the operator selects a Fedora discovered target
- **THEN** the Fedora provider SHALL plan kernel, initrd, and install-tree URLs
  through the normal provider planning path
