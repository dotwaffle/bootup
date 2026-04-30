# Boot Media Notes

Bootup should prioritize an ISO artifact before any Linux-kernel-wrapper
format. The ISO can keep shipping the same kernel plus zstd initramfs payload
used by PXE, iPXE, and vmtest.

On Ubuntu systems with `grub-imageboot` installed, an operator should be able
to place the ISO under `/boot/images/` and run `update-grub`. The package's
`/etc/grub.d/60_grub-imageboot` script is expected to discover the ISO and add
a GRUB menu entry automatically.

The ISO itself should also be directly bootable. The target shape is a hybrid
BIOS/UEFI ISO that can be burned to optical media or written directly to a USB
stick without a separate `.img` variant.

iPXE's current `genfsimg` flow is a useful reference:

- BIOS boot files are present even for EFI-capable ISOs.
- BIOS uses an El Torito no-emulation entry that boots `isolinux.bin`.
- EFI uses an alternate El Torito no-emulation entry pointing at an embedded
  FAT ESP image.
- The ESP contains `EFI/BOOT/BOOTX64.EFI` for x86_64 UEFI fallback boot.
- `xorrisofs` can emit the USB-friendly hybrid layout directly with
  `-isohybrid-gpt-basdat`; other mkisofs-compatible tools can be followed by
  `isohybrid --uefi`.

For bootup, the likely first implementation is:

- Build or reuse `dist/kernel/linux-*-bootup-amd64-bzImage`.
- Build `dist/bootup-initramfs.cpio.zst`.
- Place them under `/boot/bootup/` in the ISO filesystem.
- Add a GRUB or ISOLINUX BIOS path and a fallback x86_64 UEFI path.
- Validate with QEMU BIOS, QEMU OVMF, and Ubuntu `grub-imageboot`.
