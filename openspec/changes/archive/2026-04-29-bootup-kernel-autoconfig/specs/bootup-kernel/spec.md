## ADDED Requirements

### Requirement: Purpose-built bootup kernel configuration
Bootup SHALL document a kernel configuration suitable for launching the stage-1
environment with kernel-provided network configuration.

#### Scenario: Kernel DHCP prerequisites are documented
- **WHEN** an operator builds a bootup-oriented kernel
- **THEN** the repository SHALL identify the required built-in IPv4 DHCP
  autoconfiguration and NIC driver options for `ip=::::::dhcp`

#### Scenario: Kernel binaries are not committed
- **WHEN** the repository provides kernel build guidance
- **THEN** it SHALL avoid committing compiled kernel images or module trees

#### Scenario: Kernel config is validated locally
- **WHEN** an operator points the validation helper at a kernel config file
- **THEN** the helper SHALL report missing required built-in options and modular
  NIC options that cannot satisfy early kernel DHCP
