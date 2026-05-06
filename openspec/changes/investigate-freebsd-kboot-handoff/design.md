## Context

Bootup currently executes Linux-shaped targets through in-process kexec and can
fall back from `kexec_file_load` to u-root's Linux `kexec_load` path when the
kernel rejects an image format. That fallback is still a Linux handoff; it does
not provide FreeBSD loader metadata, memdisk behavior, or generic bootloader
emulation.

Stock FreeBSD 15.0 and mfsBSD remain deferred because they expect FreeBSD
loader semantics before the kernel starts. FreeBSD `loader.kboot` is a
FreeBSD-provided loader intended to run from a LinuxBoot-style environment, so
it is the most plausible path to investigate before designing a custom FreeBSD
handoff executor.

## Goals / Non-Goals

**Goals:**

- Prove whether `loader.kboot` can be built or obtained reproducibly for amd64.
- Prove whether it can run from bootup's Linux/u-root stage under QEMU UEFI.
- Determine how staged FreeBSD or mfsBSD artifacts must be presented to the
  loader.
- Capture a go/no-go recommendation for a future boot action implementation.

**Non-Goals:**

- Add FreeBSD or mfsBSD targets to the default catalog.
- Implement a production FreeBSD executor.
- Write FreeBSD kernel metadata, memory-map, module, or tunable handoff logic
  directly in Go.
- Commit generated FreeBSD binaries, downloaded release artifacts, initramfs
  images, or VM disk images.
- Support BIOS-only, memdisk, syslinux COM32, or ISO chainload paths in this
  spike.

## Decisions

1. Start with FreeBSD `loader.kboot` instead of a custom handoff.

   The FreeBSD kernel expects metadata that the normal Linux kexec API does not
   synthesize. A FreeBSD-maintained loader is more likely to preserve that
   contract correctly than a new bootup implementation that duplicates loader
   internals. The alternative is to implement FreeBSD loader metadata assembly
   in Go, but that should only be considered if `loader.kboot` is unavailable
   or demonstrably unsuitable.

2. Use QEMU UEFI as the first proof target.

   Recent `loader.kboot` work is aligned with LinuxBoot and UEFI. A UEFI-only
   smoke keeps the spike bounded and directly tests the path most likely to
   work on modern machines. BIOS support can be considered later if there is a
   concrete deployment need.

3. Treat mfsBSD as a useful test payload, not a separate handoff family.

   mfsBSD is valuable because it can reach a RAM-resident FreeBSD environment
   and installer workflow, but it still depends on the same FreeBSD loader
   semantics. The spike should validate either stock FreeBSD installer
   artifacts, mfsBSD artifacts, or both using the same `loader.kboot` route.

4. Keep acquisition and provenance explicit.

   The spike should record whether `loader.kboot` is available from official
   FreeBSD release artifacts, must be built from source, or needs a custom
   packaging step. Any future implementation needs a clear trust and update
   model before bootup can depend on the loader.

## Risks / Trade-offs

- `loader.kboot` may require UEFI state not available in bootup's current QEMU
  launch flow -> build a minimal QEMU UEFI reproducer before touching catalog
  code.
- The loader may not see files staged inside bootup's initramfs as a FreeBSD
  loader device -> test the file layout and mount assumptions separately from
  network fetching.
- Building `loader.kboot` may require a FreeBSD build environment or unstable
  source checkout -> document exact source revision, build command, and
  artifact hash if the spike depends on a locally built loader.
- A successful manual boot could still be too fragile for production -> require
  a reproducible script or documented command before recommending a boot action.
- Pulling this into the catalog too early would mislead operators -> keep all
  FreeBSD and mfsBSD targets out of the default catalog until the handoff has
  automated smoke evidence.
