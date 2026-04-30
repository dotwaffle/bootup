# Launching bootup

Bootup expects a stage-0 loader to load a Linux kernel and the bootup u-root
initramfs. Once Linux starts, bootup takes over target selection, verification,
staging, and kexec handoff.

The initramfs build keeps bootup's runtime payload to a single u-root
busybox-style binary. TLS roots are compiled into that binary through
`github.com/breml/rootcerts`; distro archive keyrings are not packaged by
default and must be supplied explicitly to the reusable verification hooks.
For downloaded release artifact names, checksums, manifests, and stage-0 usage
examples, see `docs/release.md`.

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

Interactive menu boots use `bootup --mode=menu --ui=auto` by default. Auto mode
uses the rich Bubble Tea terminal UI when stdin and stdout are interactive
terminals, including normal serial consoles. In the u-root busybox initramfs,
auto mode can also reopen `/dev/console` when the init command starts with
non-terminal stdio. It falls back to the plain `target> ` prompt when input or
output is redirected. Force the fallback with `--ui=plain`; use `--ui=rich`
only when a terminal is required and failure is preferable to fallback.

To smoke-test menu selection without live network assumptions:

```sh
scripts/build-initramfs.sh /tmp/bootup-current-menu-initramfs.cpio 'bootup --mode=menu --ui=auto' ''
BOOTUP_INITRAMFS=/tmp/bootup-current-menu-initramfs.cpio.zst BOOTUP_CMDLINE='console=ttyS0 panic=30' scripts/run-qemu.sh
```

On 2026-04-30, that smoke reached the rich menu under QEMU. Sending `j` then
Enter selected Ubuntu 26.04 and reached the rich planning, verifying, and
staging status output before failing at the expected network fetch step in an
isolated VM.

Current local size snapshot from the same worktree:

| Artifact | Bytes | Approx |
| --- | ---: | ---: |
| Baseline bootup binary before rich UI | 10,641,318 | 11M |
| Current bootup binary | 12,850,042 | 13M |
| Baseline raw initramfs | 10,458,064 | 10M |
| Baseline zstd initramfs | 3,336,900 | 3.2M |
| Current menu raw initramfs | 11,273,160 | 11M |
| Current menu zstd initramfs | 3,547,797 | 3.4M |

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

The default provider set also lists Ubuntu 26.04 amd64 netboot. Its boot plan
uses the official release netboot kernel and initrd:

```text
https://releases.ubuntu.com/26.04/netboot/amd64/linux
https://releases.ubuntu.com/26.04/netboot/amd64/initrd
```

Ubuntu staging uses HTTPS transport trust by default. Custom builds can supply
Ubuntu release signing key material and explicit SHA-256 hashes for the netboot
kernel/initrd if they need stronger verification.

To attempt a real QEMU boot into Ubuntu 26.04 netboot:

```sh
scripts/smoke-real-ubuntu.sh
```

The Ubuntu smoke builds a normal bootup initramfs, configures QEMU user
networking in the initramfs for host kernels without kernel DHCP support, stages
the Ubuntu netboot artifacts over HTTPS, and attempts kexec. A timeout after
the target kernel starts is expected for a manual smoke run.

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

Build a directly bootable hybrid BIOS/UEFI ISO:

```sh
scripts/build-iso.sh
```

The script discovers the current `dist/kernel/linux-*-bootup-amd64-bzImage`
and builds a menu-mode `dist/bootup-iso-initramfs.cpio.zst` when
`BOOTUP_ISO_INITRAMFS` is not set. It writes `dist/bootup.iso` by default.
It requires `grub-mkrescue`, `xorriso`, and GRUB's x86_64 EFI modules from
`grub-efi-amd64-bin` for a hybrid BIOS/UEFI artifact. Set
`BOOTUP_ISO_ALLOW_BIOS_ONLY=1` only when intentionally building a BIOS-only
local smoke artifact.

For a Debian-capable ISO, first build an initramfs with caller-supplied Debian
archive trust material, then pass it to the ISO builder:

```sh
scripts/build-debian-initramfs.sh /path/to/debian-archive-keyring.gpg dist/bootup-custom-initramfs.cpio
BOOTUP_ISO_INITRAMFS=dist/bootup-custom-initramfs.cpio.zst scripts/build-iso.sh dist/bootup-debian.iso
```

Run the ISO under QEMU BIOS:

```sh
scripts/run-qemu-iso.sh
```

Run the same image under OVMF/UEFI:

```sh
BOOTUP_QEMU_FIRMWARE=/usr/share/OVMF/OVMF_CODE_4M.fd scripts/run-qemu-iso.sh
```

No provider behavior should depend on whether bootup arrived from PXE, iPXE,
GRUB, or ISO media.

For release ISO naming, checksum verification, and the exact published artifact
set, see `docs/release.md`.
