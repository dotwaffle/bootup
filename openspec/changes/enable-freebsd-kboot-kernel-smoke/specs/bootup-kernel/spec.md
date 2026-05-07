## ADDED Requirements

### Requirement: Kernel exposes FreeBSD kboot metadata prerequisites
Bootup SHALL document and locally validate the built-in Linux kernel options
required for FreeBSD `loader.kboot` to recover the running kernel's boot
metadata from a Linux/u-root stage-1 environment.

#### Scenario: Kboot metadata options are required
- **WHEN** an operator validates a bootup-oriented amd64 kernel config
- **THEN** the validator SHALL require `CONFIG_KALLSYMS`,
  `CONFIG_KALLSYMS_ALL`, and `CONFIG_PROC_KCORE` to be built in

#### Scenario: Missing kboot metadata options are reported
- **WHEN** a kernel config omits a FreeBSD kboot metadata prerequisite or sets
  it as a module
- **THEN** the validator SHALL report the missing or modular symbol as
  unsuitable for early-stage FreeBSD kboot handoff testing
