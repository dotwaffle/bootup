## ADDED Requirements

### Requirement: Release binary build metadata
Bootup SHALL expose stamped build metadata from the release binary and publish
that metadata in the release manifest.

#### Scenario: Version command reports build metadata
- **WHEN** an operator runs `bootup --version`
- **THEN** bootup SHALL print the bootup release version, git commit, build
  date, source tree state, and Go runtime version without loading catalogs,
  provider configuration, or boot modes

#### Scenario: Release builder stamps binary metadata
- **WHEN** the release packaging workflow builds the standalone bootup binary
- **THEN** it SHALL stamp the binary with the release version, git commit,
  build date, and source tree state used for the release

#### Scenario: Manifest records binary metadata
- **WHEN** a release manifest is generated
- **THEN** it SHALL record the stamped bootup binary release version, git
  commit, build date, and source tree state

#### Scenario: Release validation checks binary metadata
- **WHEN** release validation inspects a release bundle
- **THEN** it SHALL compare the manifest's bootup binary metadata with the
  metadata reported by the release binary before publication
