# bootup-target-options Specification

## Purpose
TBD - created by archiving change add-catalog-validation-and-options. Update Purpose after archive.
## Requirements
### Requirement: Catalog target option definitions
Bootup SHALL allow static catalog targets to declare operator-selectable option
definitions.

#### Scenario: Target declares options
- **WHEN** a static catalog target declares option definitions
- **THEN** bootup SHALL expose each option with a stable ID, display label,
  option type, and command-line behavior

#### Scenario: Target omits options
- **WHEN** a static catalog target declares no option definitions
- **THEN** bootup SHALL continue to plan and boot the target without requiring
  option selection

### Requirement: Option value validation
Bootup SHALL validate selected option values before provider planning produces a
boot plan.

#### Scenario: Unknown option is selected
- **WHEN** an operator selects an option ID that the target does not declare
- **THEN** bootup SHALL fail before staging artifacts

#### Scenario: Invalid option value is selected
- **WHEN** an operator selects a value that is not valid for the declared option
  type or allowed values
- **THEN** bootup SHALL fail before staging artifacts

### Requirement: Option command-line expansion
Bootup SHALL translate selected catalog options into deterministic boot command
line additions.

#### Scenario: Option fragments are applied
- **WHEN** an operator selects valid target options
- **THEN** bootup SHALL append the resulting command-line fragments after
  provider defaults and before global operator command-line append text

#### Scenario: Option fragment is malformed
- **WHEN** a catalog option would generate a command-line fragment with invalid
  surrounding whitespace or an invalid template expansion
- **THEN** bootup SHALL reject the catalog or selected option before staging
  artifacts

### Requirement: Generic installer option coverage
Bootup SHALL support generic option definitions sufficient for common installer
parameters without hard-coding distro-specific UI.

#### Scenario: Common installer option is represented
- **WHEN** a target needs a serial console, VNC install, text install, mirror
  URL, or automated install file URL option
- **THEN** bootup SHALL be able to represent that option as catalog data when it
  can be expressed as a command-line fragment
