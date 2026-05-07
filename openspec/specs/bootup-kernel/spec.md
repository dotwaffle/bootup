# bootup-kernel Specification

## Purpose
Define the kernel configuration guidance and validation required for a
purpose-built bootup stage-1 kernel.
## Requirements
### Requirement: Purpose-built bootup kernel configuration
Bootup SHALL document a kernel configuration suitable for launching the stage-1
environment with kernel-provided network configuration.

#### Scenario: Kernel DHCP prerequisites are documented
- **WHEN** an operator builds a bootup-oriented kernel
- **THEN** the repository SHALL identify the required built-in IPv4 DHCP
  autoconfiguration and NIC driver options for `ip=::::::dhcp`

#### Scenario: Kernel boot defaults are documented
- **WHEN** an operator builds a bootup-oriented kernel
- **THEN** the repository SHALL identify kernel image compression, serial and
  framebuffer console defaults, kexec, panic reboot, and initramfs
  decompression options required for bootup

#### Scenario: Local media chainload support is documented
- **WHEN** an operator builds a bootup-oriented kernel for systems with local
  disks or removable media
- **THEN** the repository SHALL identify built-in partition, storage,
  keyboard, ext4, and VFAT options needed to inspect common local boot media

#### Scenario: Kernel binaries are not committed
- **WHEN** the repository provides kernel build guidance
- **THEN** it SHALL avoid committing compiled kernel images or module trees

#### Scenario: Latest stable kernel can be built
- **WHEN** an operator runs the repository kernel build helper without a pinned
  kernel version
- **THEN** the helper SHALL query kernel.org release metadata and build the
  latest stable upstream Linux release

#### Scenario: Kernel config is validated locally
- **WHEN** an operator points the validation helper at a kernel config file
- **THEN** the helper SHALL report missing required built-in options and modular
  NIC options that cannot satisfy early kernel DHCP

### Requirement: Kernel exposes FreeBSD kboot metadata prerequisites
Bootup SHALL document and locally validate the built-in Linux kernel options
required for FreeBSD `loader.kboot` to recover the running kernel's boot
metadata from a Linux/u-root stage-1 environment.

#### Scenario: Kboot metadata options are required
- **WHEN** an operator validates a bootup-oriented amd64 kernel config
- **THEN** the validator SHALL require `CONFIG_DEBUG_KERNEL`,
  `CONFIG_KALLSYMS`, `CONFIG_KALLSYMS_ALL`, and `CONFIG_PROC_KCORE` to be
  built in

#### Scenario: Missing kboot metadata options are reported
- **WHEN** a kernel config omits a FreeBSD kboot metadata prerequisite or sets
  it as a module
- **THEN** the validator SHALL report the missing or modular symbol as
  unsuitable for early-stage FreeBSD kboot handoff testing
