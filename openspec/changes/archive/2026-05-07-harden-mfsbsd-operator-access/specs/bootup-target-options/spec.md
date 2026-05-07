## MODIFIED Requirements

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
