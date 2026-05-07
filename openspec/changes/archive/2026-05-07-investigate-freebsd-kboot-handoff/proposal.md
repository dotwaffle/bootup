## Why

Stock FreeBSD and mfsBSD artifacts still require FreeBSD loader semantics, so
bootup cannot enable them through the current Linux kexec or Multiboot paths.
FreeBSD's `loader.kboot` may provide a narrower path forward by running a
FreeBSD-aware loader from LinuxBoot/u-root before handing off to the FreeBSD
kernel.

## What Changes

- Add a focused spike to prove whether bootup can launch FreeBSD or mfsBSD by
  executing FreeBSD `loader.kboot` from the Linux/u-root environment.
- Identify how `loader.kboot` can be built, packaged, verified, and provided to
  bootup without committing generated binary payloads.
- Validate a QEMU UEFI handoff path using staged FreeBSD or mfsBSD kernel,
  module, and root filesystem artifacts.
- Document whether this path should become a future boot action or remain
  deferred.

## Capabilities

### New Capabilities

- `bootup-freebsd-kboot-handoff`: evaluation criteria and evidence for a
  FreeBSD `loader.kboot` handoff path.

### Modified Capabilities

- None. The spike does not make BSD or mfsBSD targets executable in the default
  catalog until a later implementation change adds a proven executor.

## Impact

- Research notes and OpenSpec artifacts for FreeBSD/mfsBSD handoff support.
- Possible experimental scripts or docs used only to reproduce the spike.
- No committed FreeBSD binaries, generated initramfs images, or default catalog
  entries.
