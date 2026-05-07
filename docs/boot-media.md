# Boot Media Notes

Bootup should prioritize an ISO artifact before any Linux-kernel-wrapper
format. The ISO ships the same kernel used by PXE, iPXE, and vmtest, with a
gzip initramfs payload by default for broader early-boot compatibility.

Release artifact names, checksum verification, and iPXE/GRUB/ISO consumption
examples are documented in `docs/release.md`.

On Ubuntu systems with `grub-imageboot` installed, an operator should be able
to place the ISO under `/boot/images/` and run `update-grub`. The package's
`/etc/grub.d/60_grub-imageboot` script is expected to discover the ISO and add
a GRUB menu entry automatically.

The ISO itself should also be directly bootable. The target shape is a hybrid
BIOS/UEFI ISO that can be burned to optical media or written directly to a USB
stick without a separate `.img` variant.

Build the ISO from the current bootup kernel and a menu-mode initramfs:

```sh
scripts/build-iso.sh
```

The builder uses `grub-mkrescue` and `xorriso`. Install
`grub-efi-amd64-bin` as well as the BIOS GRUB modules so `grub-mkrescue` can
produce a hybrid BIOS/UEFI image. The script discovers the latest
`dist/kernel/linux-*-bootup-amd64-bzImage`, builds a gzip
`dist/bootup-iso-initramfs.cpio.gz` when no initramfs is supplied, and writes
`dist/bootup.iso`. Set `BOOTUP_ISO_ALLOW_BIOS_ONLY=1` only for local BIOS-only
smoke artifacts when EFI GRUB modules are absent. Override inputs with:

```sh
BOOTUP_ISO_KERNEL=dist/kernel/linux-7.0.2-bootup-amd64-bzImage \
BOOTUP_ISO_INITRAMFS=dist/bootup-custom-initramfs.cpio.zst \
scripts/build-iso.sh dist/bootup-debian.iso
```

If a VM or kernel fails in the kernel's zstd initramfs decompressor, keep the
default gzip ISO initramfs or pass a raw cpio initramfs instead. For example:

```sh
BOOTUP_ISO_INITRAMFS=dist/bootup-iso-initramfs.cpio \
scripts/build-iso.sh dist/bootup-raw-initramfs.iso
```

Run the ISO under QEMU BIOS:

```sh
scripts/run-qemu-iso.sh
```

Run with OVMF/UEFI firmware:

```sh
BOOTUP_QEMU_FIRMWARE=/usr/share/OVMF/OVMF_CODE_4M.fd scripts/run-qemu-iso.sh
```

iPXE's current `genfsimg` flow is a useful reference:

- BIOS boot files are present even for EFI-capable ISOs.
- BIOS uses an El Torito no-emulation entry that boots `isolinux.bin`.
- EFI uses an alternate El Torito no-emulation entry pointing at an embedded
  FAT ESP image.
- The ESP contains `EFI/BOOT/BOOTX64.EFI` for x86_64 UEFI fallback boot.
- `xorrisofs` can emit the USB-friendly hybrid layout directly with
  `-isohybrid-gpt-basdat`; other mkisofs-compatible tools can be followed by
  `isohybrid --uefi`.

The ISO layout places the kernel and initramfs under `/boot/bootup/` and uses
GRUB as the BIOS/UEFI bootloader. The generated GRUB entry appends
`console=tty0` so bootup userspace writes to the video console by default, and
passes `panic=30` plus `ip=::::::dhcp` to the kernel. The project kernel
fallback command line still includes serial output; when using a custom kernel
without that fallback, set `BOOTUP_ISO_CMDLINE` to add the deployment's serial
console explicitly.
