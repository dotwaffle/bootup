# bootup-catalog-conformance Specification

## Purpose
Define dry-run catalog conformance reporting for registered targets, including
plan status, boot-artifact trust posture, and explicit live/QEMU smoke coverage
classification.
## Requirements
### Requirement: Catalog conformance matrix
Bootup SHALL provide a non-interactive catalog conformance matrix for the
currently configured provider registry.

#### Scenario: Operator renders catalog matrix
- **WHEN** an operator selects the catalog matrix mode
- **THEN** bootup SHALL render each registered target with target ID, provider,
  resolved boot action, plan status, artifact trust classification, and smoke
  coverage classification

#### Scenario: Catalog matrix is hermetic
- **WHEN** bootup renders the catalog matrix
- **THEN** it SHALL NOT download boot artifacts, stage artifacts, contact
  upstream mirrors, or launch QEMU

#### Scenario: Target plan succeeds
- **WHEN** a registered target can be planned by its provider
- **THEN** the matrix SHALL report that target with a successful plan status

#### Scenario: Target plan fails
- **WHEN** a registered target cannot be planned by its provider
- **THEN** the matrix SHALL include the target and report the planning error
  without hiding other targets

### Requirement: Artifact trust classification
Bootup SHALL classify boot-artifact trust posture from the dry-run boot plan.

#### Scenario: Pinned artifacts are planned
- **WHEN** every downloadable artifact in a boot plan has a SHA-256 pin
- **THEN** the matrix SHALL classify the target as hash-pinned

#### Scenario: Signed metadata is planned
- **WHEN** a boot plan uses signature metadata for artifact verification
- **THEN** the matrix SHALL classify the target as signed-metadata backed

#### Scenario: Release metadata is planned
- **WHEN** a boot plan uses release metadata and checksums for artifact
  verification
- **THEN** the matrix SHALL classify the target as release-metadata backed

#### Scenario: HTTPS-only artifacts are planned
- **WHEN** a boot plan has downloadable artifacts without hashes or signed
  metadata and all artifact URLs use HTTPS
- **THEN** the matrix SHALL classify the target as HTTPS-only

#### Scenario: Local boot target is planned
- **WHEN** a boot plan does not download boot artifacts
- **THEN** the matrix SHALL classify artifact trust as not applicable

### Requirement: Smoke coverage classification
Bootup SHALL classify smoke coverage using explicit helper support for known
target families.

#### Scenario: Generic Linux catalog target is classified
- **WHEN** a static catalog target uses `linux-kexec` with generic Linux source
  metadata containing a base URL and kernel path
- **THEN** the matrix SHALL classify it as covered by the live catalog staging
  helper and catalog QEMU helper

#### Scenario: Dedicated distro smoke target is classified
- **WHEN** a static catalog target has a dedicated Debian, Ubuntu, or mfsBSD
  QEMU smoke helper
- **THEN** the matrix SHALL classify it with that helper's smoke coverage label

#### Scenario: Target has no live helper
- **WHEN** a static catalog target has no explicit live or QEMU smoke helper
  support
- **THEN** the matrix SHALL classify it as metadata-only
