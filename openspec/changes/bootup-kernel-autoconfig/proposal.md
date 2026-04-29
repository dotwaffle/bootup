## Why

Bootup should have a documented path for a purpose-built kernel instead of
depending on arbitrary host distro kernels. Kernel IP autoconfiguration lets the
stage-1 environment enter bootup with network state already established, while
keeping the initramfs as one u-root binary and avoiding a user-space DHCP
client.

## What Changes

- Add a kernel configuration fragment for a bootup-oriented amd64/QEMU kernel.
- Document the required built-in networking options for `ip=::::::dhcp`,
  including `CONFIG_IP_PNP_DHCP`, `e1000`, and `virtio_net`.
- Add helper validation so local builds can detect missing kernel config
  requirements before attempting a kernel-DHCP boot.
- Update QEMU launch documentation and examples to distinguish purpose-built
  kernel DHCP from the static fallback used with host kernels.

## Capabilities

### New Capabilities
- `bootup-kernel`: Kernel configuration and launch behavior for the bootup
  stage-1 environment.

### Modified Capabilities
- `bootup-netboot`: Clarifies that runtime network preparation validates
  existing network state and can consume kernel-provided DNS hints, rather than
  requiring bootup to run a DHCP client.

## Impact

- Adds documented kernel configuration assets under version control.
- Adds shell validation for local kernel config inspection.
- Updates launch and VM documentation.
- Does not rebuild or ship a kernel binary in the repository.
