## Context

The current QEMU smoke works against the host Debian kernel by including the
host `e1000` module in the initramfs and assigning QEMU user-network addresses
statically. That is useful for local smoke testing, but it is not the desired
bootup distribution shape. The preferred path is a purpose-built kernel where
the NIC driver and kernel IP autoconfiguration are built in, letting the kernel
honor `ip=::::::dhcp` before `/init` starts.

## Decisions

### Ship configuration, not kernel binaries

The repository will contain a config fragment and validation script, not a
compiled kernel. Kernel binaries are large, host/toolchain-specific artifacts
and should be produced by a separate build or release process.

### Prefer built-in drivers for DHCP boot

Kernel DHCP runs before modules from the initramfs can be loaded, so the
network driver used for the boot interface must be built in. The initial
fragment should cover QEMU-friendly and common virtualized paths: `e1000`,
`virtio_net`, PCI support, IPv4, and IP autoconfiguration with DHCP.

### Keep host-kernel smoke as fallback

The existing real Debian smoke remains useful on developer machines where
`CONFIG_IP_PNP_DHCP` is unavailable or NICs are modules. The docs should make
clear that this fallback is not the preferred production kernel path.

### Validate by inspecting config text

A small script can check a kernel config file for required `=y` options and
reject known-bad modular choices. This keeps validation fast, hermetic, and
easy to run against `/boot/config-*` or a generated `.config`.

## Risks / Trade-offs

- Kernel config symbols vary slightly across versions. The fragment should stay
  minimal and documented rather than pretending to be a complete defconfig.
- Built-in NIC drivers increase kernel size, but avoid early userspace DHCP and
  module-loading ordering problems.
- `ip=::::::dhcp` does not write DNS to `/etc/resolv.conf`; bootup must keep
  consuming `/proc/net/pnp` hints.

## Rollout

1. Add the kernel config fragment and config-check script.
2. Document the purpose-built kernel launch path and fallback smoke path.
3. Add tests or shell checks for the validation script.
4. Run normal Go checks plus shell syntax checks.
