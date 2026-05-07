## Why

mfsBSD now gives bootup a useful BSD rescue bridge, but stock FreeBSD,
OpenBSD, HDT, memdisk ISO, and syslinux COM32 flows still need bootloader or
firmware-style semantics that Linux kexec and FreeBSD `loader.kboot` do not
provide. The next step is to prove whether bootup can support that class with a
dedicated chainload handoff family, or document why it must remain deferred.

## What Changes

- Add a focused spike for chainload and target-visible-media handoff routes.
- Evaluate candidate routes for stock FreeBSD bootonly media, OpenBSD install
  media or `bsd.rd`, syslinux/memdisk-shaped payloads, and EFI/BIOS chainload
  flows.
- Build or document a QEMU proof helper only if a route can be exercised
  without committing generated binaries, ISOs, disk images, or firmware state.
- Record the evidence and recommendation before any executable default catalog
  target is added.

## Capabilities

### New Capabilities

- `bootup-chainload-handoff`: evaluation criteria, constraints, and smoke
  evidence for bootloader/firmware-style handoff routes.

### Modified Capabilities

- None. This spike does not relax the existing deferred-action contract or add
  executable stock FreeBSD, OpenBSD, memdisk, COM32, HDT, or iPXE chainload
  targets.

## Impact

- Research notes, specs, docs, and possible smoke-helper scripts for
  chainload feasibility.
- Possible local-only QEMU experiments using artifacts in `/tmp` or other
  ignored paths.
- No committed distro payloads, generated boot media, firmware variable
  images, or generated bootloader binaries.
