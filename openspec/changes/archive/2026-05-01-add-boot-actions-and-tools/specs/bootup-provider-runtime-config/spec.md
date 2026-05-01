## ADDED Requirements

### Requirement: Startup command-line append configuration
Bootup SHALL expose a startup option for appending operator-supplied kernel
parameters to selected boot plans.

#### Scenario: Append option is absent
- **WHEN** bootup starts without command-line append configuration
- **THEN** provider command lines SHALL remain unchanged

#### Scenario: Append option is present
- **WHEN** bootup starts with command-line append configuration
- **THEN** bootup SHALL append the configured text to the selected boot plan
  command line

### Requirement: Startup network configuration
Bootup SHALL expose startup options for interface, address, default gateway,
and DNS configuration.

#### Scenario: Address and gateway are supplied
- **WHEN** bootup starts with interface address and default gateway
  configuration
- **THEN** it SHALL configure the link and default route before provider
  operations

#### Scenario: DNS servers are supplied
- **WHEN** bootup starts with DNS server configuration
- **THEN** it SHALL write resolver configuration before provider operations
