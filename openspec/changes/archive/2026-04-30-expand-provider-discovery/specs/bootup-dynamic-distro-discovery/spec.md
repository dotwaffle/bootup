## ADDED Requirements

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

## MODIFIED Requirements

### Requirement: Discovery failure handling
Bootup SHALL report dynamic discovery failures without corrupting the static
catalog target list.

#### Scenario: Discovery finds no targets
- **WHEN** dynamic discovery succeeds but returns no concrete targets
- **THEN** bootup SHALL report the empty result clearly and keep the current
  stage-1 environment available for diagnosis
