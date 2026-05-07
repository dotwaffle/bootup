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

## Investigation Notes

### Loader Acquisition

FreeBSD 15.0-RELEASE amd64 ships `loader.kboot` in the official base
distribution set:

```sh
curl -fsSL https://download.freebsd.org/releases/amd64/amd64/15.0-RELEASE/base.txz |
  tar -tJf - | rg 'loader\.kboot|kboot'
```

Observed entries:

```text
./boot/loader.help.kboot
./boot/loader.kboot
```

The matching source release also includes the kboot loader source under
`usr/src/stand/kboot/`, including amd64-specific sources in
`usr/src/stand/kboot/kboot/arch/amd64/` and
`usr/src/stand/kboot/libkboot/arch/amd64/`.

The loader selected for the first spike is the shipped 15.0-RELEASE amd64
binary from `base.txz`, not a local build. Release metadata:

- Release: `15.0-RELEASE`
- Branch: `releng/15.0`
- Revision: `7aedc8de6446`
- Build date: `20251128`
- Distribution set: `base.txz`
- Distribution SHA-256 from `MANIFEST`:
  `ac0c933cc02ee8af4da793f551e4a9a15cdcf0e67851290b1e8c19dd6d30bba8`
- Local extraction path:
  `/tmp/bootup-freebsd-kboot-15.0.tRpjS6/boot/loader.kboot`
- Loader SHA-256:
  `5a8e99a002b4b4c26051005fe4957c2faf3c733dc0bd09dc265537dc831b4dec`
- Help file SHA-256:
  `c228f4b44dad94b4b4473ba441d3e8dea07a0ee9f447418aa3954117b00de172`

Extraction command:

```sh
mkdir -p /tmp/bootup-freebsd-kboot-15.0
curl -fsSL https://download.freebsd.org/releases/amd64/amd64/15.0-RELEASE/base.txz |
  tar -xJf - -C /tmp/bootup-freebsd-kboot-15.0 \
    ./boot/loader.kboot ./boot/loader.help.kboot
```

The extracted loader is an x86-64 static ELF executable:

```text
ELF 64-bit LSB executable, x86-64, version 1 (SYSV), statically linked,
stripped
```

No FreeBSD loader binaries or release payloads are tracked in the repository.

### Payload Layout

The first proof payload is the official FreeBSD 15.0-RELEASE amd64 bootonly
ISO. It is the smallest stock ISO image and includes the installer `/boot`
tree, the FreeBSD kernel, and the CD-root configuration needed by the stock
installer environment.

- Compressed payload:
  `FreeBSD-15.0-RELEASE-amd64-bootonly.iso.xz`
- Compressed SHA-256:
  `f7a3698ead2ae1ac9ac374bda32bd1bf9e31edbe0d94ee25a2dee13b0af0d165`
- Uncompressed payload:
  `FreeBSD-15.0-RELEASE-amd64-bootonly.iso`
- Uncompressed SHA-256:
  `78b40ce8065fcc08bfef96c05c5cbfaaa996059130134f5b097389df41847b46`
- Local payload path:
  `/tmp/bootup-freebsd-kboot-15.0.tRpjS6/FreeBSD-15.0-RELEASE-amd64-bootonly.iso`
- Extracted inspection path:
  `/tmp/bootup-freebsd-bootonly-15.0.g8gIAs`

Relevant files from the ISO:

```text
boot/defaults/loader.conf  9997 bytes
boot/kernel/kernel         29339232 bytes
boot/loader.conf           116 bytes
boot/loader.help.kboot     13653 bytes
boot/loader.kboot          492512 bytes
etc/fstab                  51 bytes
```

The kernel hash from the ISO is:

```text
b7396889dd6c268246f781a7176d1dc55861646040c5ef74686904caa9781b57  boot/kernel/kernel
```

The ISO `boot/loader.conf` contains:

```text
vfs.mountroot.timeout="10"
kernels_autodetect="NO"
loader_brand="install"
loader_menu_multi_user_prompt="Installer"
```

The ISO `etc/fstab` expects the installer root to remain available as a CD
device:

```text
/dev/iso9660/15_0_RELEASE_AMD64_BO / cd9660 ro 0 0
```

There is no separate `/boot/mfsroot` in this bootonly ISO. The first QEMU proof
therefore should attach the bootonly ISO as a CD-ROM/block device visible to
both Linux and FreeBSD, instead of copying only `/boot/kernel/kernel` into the
initramfs. Copying only kernel files would let `loader.kboot` load the kernel
from `host:/...`, but the FreeBSD installer would likely stop at mountroot
because the expected `cd9660` root device would be absent.

`loader.kboot` can discover payloads in two useful ways:

- The `host` filesystem maps loader paths such as `host:/...` to mounted Linux
  filesystems. `hostfs_root` defaults to `/`, with `/sys` and `/proc` passed
  through to the Linux host.
- The host disk path scans `/sys/block`, exposes Linux block devices as
  `/dev/<name>:` loader devices, and can select a boot device by probing for a
  FreeBSD `/boot` layout. An explicit `bootdev` environment variable can
  override auto-detection.

For this payload, the preferred first command shape is to attach the bootonly
ISO to QEMU, boot Linux/u-root with `loader.kboot` in the initramfs, and run
`loader.kboot` with `bootdev` pointing at the Linux CD-ROM block device if
auto-detection does not select it.

### QEMU Proof

The proof used the extracted FreeBSD loader as an extra file overlay in a
temporary bootup initramfs:

```sh
mkdir -p /tmp/bootup-freebsd-kboot-extra.36H9cX/bin \
  /tmp/bootup-freebsd-kboot-extra.36H9cX/boot
cp /tmp/bootup-freebsd-kboot-15.0.tRpjS6/boot/loader.kboot \
  /tmp/bootup-freebsd-kboot-extra.36H9cX/bin/loader.kboot
cp /tmp/bootup-freebsd-kboot-15.0.tRpjS6/boot/loader.help.kboot \
  /tmp/bootup-freebsd-kboot-extra.36H9cX/boot/loader.help.kboot

GOCACHE=/tmp/bootup-go-build-cache-freebsd-kboot \
BOOTUP_INITRAMFS_ZSTD=/tmp/bootup-freebsd-kboot-initramfs.cpio.zst \
scripts/build-initramfs.sh /tmp/bootup-freebsd-kboot-initramfs.cpio \
  gosh '' /tmp/bootup-freebsd-kboot-extra.36H9cX:/
```

The repository Go build cache had stale entries during the run, so the proof
used the private `GOCACHE` above instead of mutating the shared cache. The
generated initramfs was not committed.

The generated initramfs contained:

```text
bin/loader.kboot
boot/loader.help.kboot
bbin/gosh
bin/uinit
```

The zstd initramfs was built at
`/tmp/bootup-freebsd-kboot-initramfs.cpio.zst`, with SHA-256:

```text
03dc7bb666f1d0b1d30915f877212399d7d64830f9c6feea9af82b936f0c56be
```

The proof ISO was built with:

```sh
BOOTUP_ISO_INITRAMFS=/tmp/bootup-freebsd-kboot-initramfs.cpio.zst \
BOOTUP_ISO_CMDLINE='console=ttyS0,115200n8 panic=30' \
scripts/build-iso.sh /tmp/bootup-freebsd-kboot.iso
```

The first UEFI attempt attached the FreeBSD bootonly ISO as an IDE CD-ROM:

```sh
qemu-system-x86_64 -m 2048 -nographic -no-reboot \
  -drive if=pflash,format=raw,unit=0,readonly=on,file=/usr/share/OVMF/OVMF_CODE_4M.fd \
  -drive if=pflash,format=raw,unit=1,file=/tmp/bootup-freebsd-kboot-ovmf-vars.fd \
  -drive if=none,id=bootupcd,media=cdrom,readonly=on,file=/tmp/bootup-freebsd-kboot.iso \
  -device ide-cd,drive=bootupcd,bootindex=1 \
  -drive if=none,id=freebsdcd,media=cdrom,readonly=on,file=/tmp/bootup-freebsd-kboot-15.0.tRpjS6/FreeBSD-15.0-RELEASE-amd64-bootonly.iso \
  -device ide-cd,drive=freebsdcd
```

Linux reached the u-root shell and the loader files were present, but no
`/dev/sr0` or `/dev/sr1` devices appeared. The generated bootup kernel config
has `# CONFIG_BLK_DEV_SR is not set`, so this failure only proved that CD-ROM
payload presentation is not available with the current kernel config.

The second UEFI attempt attached the FreeBSD bootonly ISO as a read-only
virtio block device:

```sh
cp /usr/share/OVMF/OVMF_VARS_4M.fd /tmp/bootup-freebsd-kboot-ovmf-vars.fd
qemu-system-x86_64 -m 2048 -nographic -no-reboot \
  -drive if=pflash,format=raw,unit=0,readonly=on,file=/usr/share/OVMF/OVMF_CODE_4M.fd \
  -drive if=pflash,format=raw,unit=1,file=/tmp/bootup-freebsd-kboot-ovmf-vars.fd \
  -drive if=none,id=bootupcd,media=cdrom,readonly=on,file=/tmp/bootup-freebsd-kboot.iso \
  -device ide-cd,drive=bootupcd,bootindex=1 \
  -drive if=none,id=freebsddisk,format=raw,readonly=on,file=/tmp/bootup-freebsd-kboot-15.0.tRpjS6/FreeBSD-15.0-RELEASE-amd64-bootonly.iso \
  -device virtio-blk-pci,drive=freebsddisk
```

Linux detected the payload:

```text
virtio_blk virtio0: [vda] 1086436 512-byte logical blocks (556 MB/530 MiB)
vda: vda1 vda2
```

From the u-root shell, the manual loader command was:

```sh
bootdev=/dev/vda: /bin/loader.kboot
```

`loader.kboot` executed, scanned the block device, selected `/dev/vda:`, loaded
the FreeBSD loader configuration, displayed the stock FreeBSD installer loader
menu, loaded `/boot/kernel/kernel`, and loaded configured modules. The key
observed lines were:

```text
Trying /dev/vda:
Boot device: /dev/vda: with hostfs_root /
FreeBSD/amd64 kboot loader, Revision 3.0
Can't find symbol boot_params
Populate worked...
Loading /boot/defaults/loader.conf
Loading /boot/loader.conf
Welcome to FreeBSD
Loading kernel...
/boot/kernel/kernel text=0x1865e8
Loading configured modules...
Start @ 0xffffffff80387000 ...
panic: Can't get UEFI memory map, nor a pointer to it, can't proceed.
```

A legacy BIOS isolation run used the same proof ISO and virtio-block payload,
without OVMF:

```sh
qemu-system-x86_64 -m 2048 -nographic -no-reboot \
  -drive if=none,id=bootupcd,media=cdrom,readonly=on,file=/tmp/bootup-freebsd-kboot.iso \
  -device ide-cd,drive=bootupcd,bootindex=1 \
  -drive if=none,id=freebsddisk,format=raw,readonly=on,file=/tmp/bootup-freebsd-kboot-15.0.tRpjS6/FreeBSD-15.0-RELEASE-amd64-bootonly.iso \
  -device virtio-blk-pci,drive=freebsddisk
```

It reached the same `loader.kboot` menu and failed with the same
`boot_params`/UEFI memory map panic. That isolates the blocker away from OVMF
firmware state and away from payload visibility.

The relevant FreeBSD 15.0 `loader.kboot` source path is
`usr/src/stand/kboot/kboot/arch/amd64/load_addr.c`. On amd64 it reads Linux's
`boot_params` data through `/proc/kallsyms` and `/proc/kcore` so it can recover
the EFI system table and memory-map pointer, then `efi_bi_loadsmap` adds
`MODINFOMD_EFI_MAP` metadata for the FreeBSD kernel. The current bootup kernel
does not expose the required `boot_params` symbol to the loader and has
`# CONFIG_PROC_KCORE is not set`, so `loader.kboot` cannot construct the
FreeBSD EFI memory-map metadata and intentionally panics before `kexec_load`.

### Recommendation

Do not add FreeBSD or mfsBSD targets to the executable default catalog yet.
The useful result is narrower: FreeBSD 15.0's shipped `loader.kboot` is
available, can run from bootup's Linux/u-root stage, and can read the stock
FreeBSD installer ISO when it is exposed as a Linux block device. The current
blocker is the FreeBSD loader's Linux-kernel metadata dependency, not basic
loader execution or FreeBSD payload layout.

A follow-up `freebsd-kboot` implementation change should start by producing a
kboot-compatible bootup kernel smoke artifact. At minimum, that means enabling
and validating the Linux kernel interfaces `loader.kboot` uses to read
`boot_params`, including `/proc/kcore`, and confirming that `/proc/kallsyms`
contains the `boot_params` data symbol. The follow-up should keep the payload
presentation as a block device, either by continuing to use a raw virtio-block
ISO attachment or by explicitly adding CD-ROM block support if CD devices are
needed.

Only after a QEMU UEFI smoke reaches the FreeBSD installer or a mfsBSD shell
should bootup grow a production `freebsd-kboot` boot action or catalog entries.
Until then, FreeBSD 15.0, mfsBSD, and other FreeBSD-loader-shaped payloads
remain deferred.
