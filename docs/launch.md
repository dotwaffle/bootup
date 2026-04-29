# Launching bootup

Bootup expects a stage-0 loader to load a Linux kernel and the bootup u-root
initramfs. Once Linux starts, bootup takes over target selection, verification,
staging, and kexec handoff.

The initramfs build keeps bootup's runtime payload to a single u-root
busybox-style binary. TLS roots are compiled into that binary through
`github.com/breml/rootcerts`; distro archive keyrings are not packaged by
default and must be supplied explicitly to the reusable verification hooks.

## QEMU

Build the initramfs. The script writes both a raw cpio and a zstd-compressed
initramfs:

```sh
scripts/build-initramfs.sh
```

The script also accepts an output path, a uinit command, and optional Go build
tags:

```sh
scripts/build-initramfs.sh dist/bootup-initramfs.cpio 'bootup --mode=menu --prepare-runtime' ''
```

Run with a local kernel:

```sh
scripts/run-qemu.sh
```

Override the kernel, initramfs, or kernel command line with `BOOTUP_KERNEL`,
`BOOTUP_INITRAMFS`, and `BOOTUP_CMDLINE`. The default command line includes
`panic=30` so kernel panics remain visible briefly and then reboot.
The initramfs build runs `bootup --hold` by default so smoke-test boots do not
exit PID 1 after printing the target list; override it with `BOOTUP_UINITCMD`.

For a Debian-capable single binary, generate ignored Go source from a local
OpenPGP public keyring before building the initramfs:

```sh
go run ./cmd/bootup-keyring-source -o internal/trustmaterial/debian_archive_keyring_generated.go /usr/share/keyrings/debian-archive-keyring.gpg
scripts/build-initramfs.sh dist/bootup-initramfs.cpio 'bootup --mode=menu --prepare-runtime' ''
```

## iPXE

`examples/bootup.ipxe` shows the minimal shape:

```text
kernel http://boot.example/bootup/vmlinuz console=ttyS0 panic=30
initrd http://boot.example/bootup/bootup-initramfs.cpio.zst
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
