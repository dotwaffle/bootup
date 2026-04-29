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

The script also accepts an output path, a uinit command, optional Go build
tags, and optional extra files:

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
Purpose-built bootup kernels should also include `ip=::::::dhcp` so the kernel
configures networking before bootup starts.

For a Debian-capable single binary, generate ignored Go source from a local
OpenPGP public keyring before building the initramfs:

```sh
go run ./cmd/bootup-keyring-source -o internal/trustmaterial/debian_archive_keyring_generated.go /usr/share/keyrings/debian-archive-keyring.gpg
scripts/build-initramfs.sh dist/bootup-initramfs.cpio 'bootup --mode=menu --prepare-runtime' ''
```

`--prepare-runtime` does not run a user-space DHCP client. Network addressing
should already be provided by the kernel command line, the boot loader, or the
initramfs command used by a local smoke helper. With a purpose-built bootup
kernel, prefer kernel autoconfiguration: build the NIC driver into the kernel,
enable `CONFIG_IP_PNP_DHCP`, and append `ip=::::::dhcp`. DNS servers learned
by the kernel are exposed through `/proc/net/pnp`; bootup copies those hints
into `/etc/resolv.conf` when that file is absent. See `docs/kernel.md` for the
kernel config fragment and validator.

The helper below performs the same build and removes the ignored generated
source after the initramfs has been created:

```sh
scripts/build-debian-initramfs.sh /usr/share/keyrings/debian-archive-keyring.gpg
```

To attempt a real QEMU boot into Debian Installer:

```sh
scripts/smoke-real-debian.sh /usr/share/keyrings/debian-archive-keyring.gpg
```

Expected local failure modes:

- Missing or unreadable keyring: the helper exits before building.
- No network in the VM: bootup reports route, DNS, TLS, or fetch failures.
- Kernel NIC driver is modular and unavailable: the smoke helper tries to
  include and load the host `e1000` module for QEMU user networking.
- The host kernel used by the helper may not provide DNS/route state through
  kernel autoconfiguration, because `CONFIG_IP_PNP` can be unset and QEMU NIC
  drivers can be modules. The smoke helper therefore configures
  `10.0.2.15/24` directly, then sets the expected `10.0.2.2` default route and
  `10.0.2.3` resolver before starting bootup.
- Missing QEMU or kernel: the smoke script exits before or during VM launch.
- kexec blocked by the platform: bootup renders a failure screen and leaves the
  stage-1 environment available for diagnosis.

## iPXE

`examples/bootup.ipxe` shows the minimal shape:

```text
kernel http://boot.example/bootup/vmlinuz ip=::::::dhcp console=ttyS0 panic=30
initrd http://boot.example/bootup/bootup-initramfs.cpio.zst
boot
```

The URLs should point at the stage-1 kernel and initramfs produced for the
environment.

## GRUB

`examples/grub.cfg` contains a matching menu entry:

```text
linux /bootup/vmlinuz ip=::::::dhcp console=ttyS0 panic=30
initrd /bootup/bootup-initramfs.cpio
```

## ISO

An ISO delivery path should place the same kernel and initramfs on the image
and configure ISOLINUX, GRUB, or another ISO bootloader to load them. No
provider behavior should depend on whether bootup arrived from PXE, iPXE,
GRUB, or ISO media.
