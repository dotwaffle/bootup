# Launching bootup

Bootup expects a stage-0 loader to load a Linux kernel and the bootup u-root
initramfs. Once Linux starts, bootup takes over target selection, verification,
staging, and kexec handoff.

The initramfs build keeps bootup's runtime payload to a single u-root
busybox-style binary. TLS roots are compiled into that binary through
`github.com/breml/rootcerts`; distro archive keyrings are not packaged by
default and must be supplied explicitly to the reusable verification hooks.

## QEMU

Build the initramfs:

```sh
scripts/build-initramfs.sh
```

Run with a local kernel:

```sh
scripts/run-qemu.sh
```

Override the kernel or initramfs path with `BOOTUP_KERNEL` and
`BOOTUP_INITRAMFS`.

## iPXE

`examples/bootup.ipxe` shows the minimal shape:

```text
kernel http://boot.example/bootup/vmlinuz console=ttyS0
initrd http://boot.example/bootup/bootup-initramfs.cpio
boot
```

The URLs should point at the stage-1 kernel and initramfs produced for the
environment.

## GRUB

`examples/grub.cfg` contains a matching menu entry:

```text
linux /bootup/vmlinuz console=ttyS0
initrd /bootup/bootup-initramfs.cpio
```

## ISO

An ISO delivery path should place the same kernel and initramfs on the image
and configure ISOLINUX, GRUB, or another ISO bootloader to load them. No
provider behavior should depend on whether bootup arrived from PXE, iPXE,
GRUB, or ISO media.
