## Why

PXE, iPXE, GRUB, and ISO boot paths are useful delivery mechanisms, but their
menu and scripting environments are too constrained for dynamic operating
system selection, release discovery, artifact verification, and rich local
interaction. Bootup should provide a real Linux/u-root stage-1 environment that
can make trustworthy boot decisions before handing off to a target installer.

## What Changes

- Introduce bootup as a chainloaded Linux/u-root environment that can be
  launched from PXE, iPXE, GRUB, or ISO media.
- Add a provider model for build-time Go modules that discover, plan, verify,
  and boot operating system installers.
- Implement an MVP provider for Debian trixie amd64 netboot installation.
- Require downloaded boot artifacts to be verified before kexec.
- Provide a plain serial-capable menu as the primary MVP interface.
- Define framebuffer rendering as a supported interface direction, while
  keeping it outside the critical MVP boot path.

## Capabilities

### New Capabilities

- `bootup-netboot`: Chainloadable stage-1 boot environment, provider contract,
  verified artifact staging, and kexec handoff behavior.

### Modified Capabilities

- None.

## Impact

- Adds bootup build and runtime architecture around u-root, Linux initramfs
  contents, networking, trust roots, release providers, and kexec.
- Introduces Debian archive signature and checksum verification as part of the
  boot path.
- Adds QEMU/vmtest-based integration expectations for booting the stage-1
  environment and verifying the Debian handoff path.
- Establishes future extension points for additional distributions, framebuffer
  UI, and alternate launch media without changing the MVP contract.
