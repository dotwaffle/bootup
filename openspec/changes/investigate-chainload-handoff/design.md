## Context

Bootup currently has three executable handoff shapes: Linux kexec, u-root local
boot, and FreeBSD `loader.kboot` for mfsBSD memory-root payloads. That covers
Linux netboot installers and the mfsBSD rescue bridge, but it does not cover
payloads that expect a bootloader or firmware to load them. Stock FreeBSD
bootonly media still expects target-visible `cd9660` root media after the
kernel starts. OpenBSD install flows expect OpenBSD boot blocks, `boot`,
`cdboot`, `pxeboot`, or firmware media to load `bsd.rd`. syslinux COM32,
memdisk, HDT, and iPXE chainload flows have similar bootloader-specific
assumptions.

The repository already documents these targets as deferred. This change should
turn that deferral into a sharper decision: either identify a viable executor
family with smoke evidence, or document the next blocker clearly enough to
avoid misleading catalog entries.

## Goals / Non-Goals

**Goals:**

- Identify the most plausible chainload or target-visible-media handoff route
  for bootup's Linux/u-root stage-1 environment.
- Test at least one representative route far enough to classify the blocker as
  loader acquisition, firmware state, media visibility, kernel handoff, or
  target-environment startup.
- Preserve a reproducible command, script, or documented procedure for any
  useful proof.
- Keep generated bootloader binaries, distro payloads, ISOs, disk images, and
  firmware variable images out of the repository.

**Non-Goals:**

- Do not add executable stock FreeBSD, OpenBSD, memdisk, COM32, HDT, or iPXE
  chainload targets to the default catalog.
- Do not implement a production chainload executor until the spike proves the
  route and defines its safety boundaries.
- Do not replace the mfsBSD `freebsd-kboot` rescue target.
- Do not depend on host-specific persistent firmware changes for normal bootup
  operation.

## Decisions

1. Treat this as a handoff-family spike, not a provider change.

   The common problem is not target metadata; it is the handoff mechanism.
   Adding more provider targets before the executor exists would produce
   catalog entries that cannot boot. The spike should start from boot action
   semantics and only later feed proven routes back into providers.

2. Prefer a route that survives the kernel jump without Linux-only devices.

   FreeBSD showed that a Linux hostfs or loop mount can be enough for a loader
   to read files, but not enough for the target kernel to mount root media.
   Candidate routes must explain what the target sees after handoff: firmware
   boot service, real block device, RAM disk, network boot state, or another
   target-native mechanism.

3. Keep proof artifacts local and reproducible.

   Any helper may download or build artifacts into `/tmp` or an operator
   supplied path, but committed content should be scripts, docs, tests, and
   specs only. If a route needs a bootloader binary, the source package,
   version, build command, and hash of the local output must be recorded.

4. Stop at the first hard blocker with evidence.

   A failed route is useful if it identifies the failing boundary. The spike
   should avoid broad refactors or speculative abstractions until a route has
   crossed at least one target-environment marker in QEMU.

## Risks / Trade-offs

- Chainload may require firmware behavior that cannot be driven from a
  Linux/u-root process → capture the exact firmware dependency and keep the
  target family deferred.
- A route may work only under QEMU because the test attaches extra media from
  the outside → classify it as a lab proof, not a production bootup executor.
- Bootloader binaries can introduce licensing and supply-chain concerns →
  keep them untracked and document acquisition or build provenance.
- OpenBSD and FreeBSD may need different executor families despite both being
  BSD-adjacent → keep the candidate matrix explicit instead of forcing one
  abstraction too early.
