# bootup-target-options Specification

## Purpose
Define catalog-declared target options that validate operator-selected values
and translate them into deterministic Linux command-line fragments or
action-specific boot arguments.
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
line or boot action argument additions.

#### Scenario: Linux option fragments are applied
- **WHEN** an operator selects valid target options for a Linux kexec target
- **THEN** bootup SHALL append the resulting command-line fragments after
  provider defaults and before global operator command-line append text

#### Scenario: FreeBSD kboot option fragments are applied
- **WHEN** an operator selects valid target options for a `freebsd-kboot` target
- **THEN** bootup SHALL append the resulting fragments to the `loader.kboot`
  argument list used by that target

#### Scenario: Option fragment is malformed
- **WHEN** a catalog option would generate a command-line or boot action
  argument fragment with invalid surrounding whitespace or an invalid template
  expansion
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

### Requirement: Target option secret boundary
Bootup SHALL treat catalog target options as non-secret boot argument data
unless a separate secret-safe delivery capability explicitly handles the secret
input outside option command-line expansion.

#### Scenario: Secret target option is rejected
- **WHEN** a static catalog target declares an option with a secret marker
- **THEN** bootup SHALL reject that target before rendering, planning, staging,
  or handoff

#### Scenario: Non-secret option output remains inspectable
- **WHEN** an operator selects a valid non-secret target option
- **THEN** bootup SHALL continue to render the resulting boot command-line or
  boot action argument data in diagnostics

#### Scenario: Secret delivery uses a separate capability
- **WHEN** a target needs a password, password hash, SSH key, token, or other
  secret input
- **THEN** that input MUST use a secret delivery declaration and MUST NOT be
  represented as a current target option command-line or loader-argument
  fragment
