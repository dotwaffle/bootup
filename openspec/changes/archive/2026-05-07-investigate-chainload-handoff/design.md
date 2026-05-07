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

## Candidate Matrix

| Candidate | Required boot semantics | Target-visible state | Current fit |
| --- | --- | --- | --- |
| Stock FreeBSD bootonly ISO | FreeBSD loader semantics plus target-visible `cd9660` root media | The ISO must appear as a FreeBSD-visible block device after the kernel jump | Best first proof because `loader.kboot` and a QEMU helper already exist |
| mfsBSD memory-root ISO | FreeBSD `loader.kboot` with `mfsroot` preloaded from hostfs | `md0` RAM root; no target-visible payload disk needed | Already productized as `mfsbsd-142-amd64`; not a chainload proof |
| OpenBSD `bsd.rd` | OpenBSD boot blocks, `boot`, `cdboot`, `pxeboot`, or firmware media load an OpenBSD kernel | OpenBSD kernel entry state plus any firmware/media state expected by `bsd.rd` | Not Linux kexec or Multiboot; local check shows gzip-compressed ELF64 with no Multiboot header |
| syslinux COM32, HDT, memdisk | syslinux/isolinux runtime and BIOS-style module APIs | COM32 ABI, BIOS interrupt model, or memdisk-presented RAM disk | Does not fit Linux kexec; likely needs a real syslinux/firmware environment |
| Generic EFI chainload | Firmware launches an EFI application from a block device or ESP | UEFI boot services, device paths, and possibly NVRAM variables | Possible research route, but u-root's local boot path does not execute arbitrary EFI apps |
| u-root experimental `uefiboot` | kexec into an EDK2 UefiPayloadPkg firmware volume | UEFI payload config built from Linux memory map, ACPI, SMBIOS, and serial data | Interesting but not a direct ISO/BOOTX64.EFI chainloader |

## Selected First Route

The first candidate is the stock FreeBSD bootonly proof using
`scripts/smoke-freebsd-kboot.sh`. It is the least invasive useful route because
it reuses the proven FreeBSD `loader.kboot` acquisition path and directly tests
the missing product capability: whether target-visible ISO media can carry the
installer past the kernel jump. A successful QEMU smoke would still be
lab-only if the FreeBSD ISO is attached by QEMU from outside bootup, but it
would prove the remaining blocker is product media presentation rather than
loader execution.

## Local Facility Notes

- `github.com/u-root/u-root/pkg/boot` is a high-level kexec interface for
  booting another OS from Linux. It covers Linux images and supported
  Multiboot images, not firmware chainloading.
- `github.com/u-root/u-root/pkg/boot/multiboot` crafts `kexec_load` segments
  for Multiboot kernels. It does not make non-Multiboot OpenBSD or syslinux
  COM32 payloads bootable.
- The bundled u-root `boot` command looks for local bootable block devices and
  parses Linux-oriented boot configuration; it replaces a traditional
  bootloader for supported local OS entries, but is not a generic EFI app,
  BIOS bootsector, COM32, or memdisk executor.
- u-root's experimental `uefiboot` command loads an EDK2 UefiPayloadPkg
  firmware volume through kexec. That may be worth a later spike, but it is not
  the same as asking current firmware to boot an arbitrary attached ISO or ESP.

## Stock FreeBSD Proof Result

On 2026-05-07, the selected stock FreeBSD route reached the installer prompt
after the FreeBSD kernel jump.

The reproducible smoke command was:

```sh
BOOTUP_FREEBSD_KBOOT_TIMEOUT=150 \
BOOTUP_FREEBSD_KBOOT_LOADER=/tmp/bootup-freebsd-kboot-smoke.5FGhWw/freebsd-base/boot/loader.kboot \
BOOTUP_FREEBSD_KBOOT_HELP=/tmp/bootup-freebsd-kboot-smoke.5FGhWw/freebsd-base/boot/loader.help.kboot \
BOOTUP_FREEBSD_KBOOT_ISO=/tmp/bootup-freebsd-kboot-smoke.5FGhWw/FreeBSD-15.0-RELEASE-amd64-bootonly.iso \
BOOTUP_FREEBSD_KBOOT_KERNEL=dist/kernel/linux-7.0.3-bootup-amd64-bzImage \
BOOTUP_FREEBSD_KBOOT_KERNEL_CONFIG=dist/kernel/linux-7.0.3-bootup-amd64.config \
scripts/smoke-freebsd-kboot.sh
```

The same helper can discover and download the official FreeBSD artifacts
itself. The default artifact URLs are:

```text
base.txz https://download.freebsd.org/releases/amd64/amd64/15.0-RELEASE/base.txz
bootonly.iso.xz https://download.freebsd.org/releases/amd64/amd64/ISO-IMAGES/15.0/FreeBSD-15.0-RELEASE-amd64-bootonly.iso.xz
```

Artifact provenance for the proof:

- `loader.kboot` and `loader.help.kboot` came from FreeBSD 15.0-RELEASE
  `base.txz`.
- `FreeBSD-15.0-RELEASE-amd64-bootonly.iso` came from the official amd64
  `ISO-IMAGES/15.0` directory.
- The bootup kernel was the local generated
  `dist/kernel/linux-7.0.3-bootup-amd64-bzImage`, with
  `dist/kernel/linux-7.0.3-bootup-amd64.config` prevalidated by
  `scripts/check-kernel-config.sh`.
- OVMF code came from the host `/usr/share/OVMF/OVMF_CODE_4M.fd`; the mutable
  VARS image was copied into the `/tmp/bootup-freebsd-kboot-smoke.*` work
  directory.
- Generated initramfs, proof ISO, OVMF VARS copy, serial logs, and downloaded
  payloads stayed under `/tmp/bootup-freebsd-kboot-smoke.*`.

The QEMU proof attached the FreeBSD bootonly ISO as a read-only virtio block
device. Linux mounted that device at `/mnt/freebsd` so `loader.kboot` could
load the kernel through `hostfs_root=/mnt/freebsd`, and the same QEMU-attached
device survived the kernel replacement as `vtbd0`.

Relevant serial markers from
`/tmp/bootup-freebsd-kboot-smoke.r1nhQA/qemu.log`:

```text
FreeBSD/amd64 kboot loader, Revision 3.0
UEFI SYSTAB PA: 0x7f5ec018
Start @ 0xffffffff80387000 ...
---<<BOOT>>---
Trying to mount root from cd9660:/dev/iso9660/15_0_RELEASE_AMD64_BO [ro]...
Dual Console: Serial Primary, Video Secondary
Starting primary installer on ttyu0
Welcome to FreeBSD!
Console type [vt100]:
```

The helper exited successfully with:

```text
FreeBSD kboot smoke reached target marker after kernel jump.
```

Classification: lab-only viable. The route proves `loader.kboot` can boot the
stock FreeBSD 15.0 installer when equivalent ISO media is target-visible after
handoff. It does not yet prove a production bootup executor, because bootup
cannot currently create a real block device that remains visible to the
replacement FreeBSD kernel on bare metal. The production blocker is target
media presentation, not loader acquisition, serial console configuration, or
FreeBSD kernel startup.

Recommendation: keep stock FreeBSD installer targets out of the executable
default catalog until bootup has a production target-visible media mechanism.
The next useful implementation spike should focus on that media mechanism for
FreeBSD-like installer ISOs. OpenBSD, syslinux COM32, HDT, memdisk, and generic
EFI chainload remain deferred because this proof did not exercise their
bootloader or firmware ABIs.

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
