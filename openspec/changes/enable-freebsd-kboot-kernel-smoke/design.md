## Context

The archived FreeBSD kboot spike proved three useful facts:

- FreeBSD 15.0-RELEASE amd64 ships `loader.kboot` in `base.txz`.
- The loader can run from bootup's Linux/u-root environment.
- The loader can find the stock FreeBSD bootonly ISO when QEMU exposes that
  ISO as a virtio block device.

The spike did not reach the FreeBSD installer. The loader printed
`Can't find symbol boot_params` and later panicked with:

```text
panic: Can't get UEFI memory map, nor a pointer to it, can't proceed.
```

The relevant FreeBSD source is
`usr/src/stand/kboot/kboot/arch/amd64/load_addr.c`. On amd64,
`loader.kboot` reads Linux's `boot_params` data through `/proc/kallsyms` and
`/proc/kcore`, then adds `MODINFOMD_EFI_MAP` metadata before handing off to the
FreeBSD kernel. The current bootup kernel config has `CONFIG_KALLSYMS=y`, but
does not have `CONFIG_KALLSYMS_ALL=y` and has `# CONFIG_PROC_KCORE is not set`.

## Goals / Non-Goals

**Goals:**

- Make the bootup amd64 kernel fragment expose the Linux interfaces that
  FreeBSD `loader.kboot` expects for metadata recovery.
- Teach the kernel config validator to fail when those prerequisites are
  absent or modular.
- Add a reproducible, opt-in FreeBSD kboot smoke procedure that reuses
  temporary `/tmp` artifacts and verifies whether the prior metadata blocker is
  cleared.
- Keep the smoke result explicit: either reaching the FreeBSD installer/mfsBSD
  shell or reporting the next blocker with serial evidence.

**Non-Goals:**

- Add a production `freebsd-kboot` boot action.
- Add FreeBSD or mfsBSD targets to the executable default catalog.
- Commit FreeBSD binaries, release payloads, generated kernels, initramfs
  images, ISOs, or VM disks.
- Replace the block-device payload proof with memdisk, BIOS chainload, or ISO
  emulation.

## Decisions

1. Add kernel prerequisites to the existing bootup amd64 fragment.

   `loader.kboot` needs the running Linux kernel to expose the `boot_params`
   data symbol and allow reading it from `/proc/kcore`. That makes
   `CONFIG_KALLSYMS_ALL=y` and `CONFIG_PROC_KCORE=y` kernel prerequisites for
   the FreeBSD kboot smoke. Keeping these in the shared fragment also makes the
   Docker-built test kernel and release-oriented bootup kernel converge.

2. Validate the prerequisites in `scripts/check-kernel-config.sh`.

   The Docker kernel build already runs this validator after `make
   olddefconfig`, and the repository already has hermetic tests for missing and
   modular symbols. Extending that path catches regressions without requiring a
   VM or network test.

3. Use a dedicated opt-in smoke script for FreeBSD kboot.

   The smoke needs external FreeBSD release artifacts, QEMU, OVMF, and enough
   time to boot through the FreeBSD loader. A script makes the manual proof
   reproducible while keeping it out of default CI. The script should download
   or consume caller-supplied `loader.kboot` and FreeBSD ISO artifacts, stage
   them in `/tmp`, build a temporary bootup initramfs/ISO, attach the FreeBSD
   ISO as a read-only virtio block device, and run the loader with
   `bootdev=/dev/vda:`.

4. Treat installer reach as the success signal.

   Passing the prior metadata point is useful but not enough. The smoke should
   only report success when serial output reaches a FreeBSD installer prompt or
   mfsBSD shell marker. If the loader gets past metadata but fails at mountroot
   or device discovery, that should be recorded as the next blocker.

## Risks / Trade-offs

- `CONFIG_KALLSYMS_ALL` and `CONFIG_PROC_KCORE` expose more kernel information
  from the stage-1 environment -> keep this scoped to the purpose-built bootup
  kernel and document why these knobs exist.
- The loader may still fail after metadata recovery because the FreeBSD kernel
  cannot mount the ISO root from the same virtio block device -> keep the
  smoke's failure capture explicit and do not add catalog entries until the
  installer or mfsBSD shell is observed.
- Building a new kernel can require Docker and network access -> keep unit
  validation hermetic, and keep the QEMU smoke opt-in.
- The smoke depends on FreeBSD mirror availability -> support caller-supplied
  artifact paths and document checksums so repeat runs can avoid fresh network
  downloads.
