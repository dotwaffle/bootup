# Launching bootup

Bootup expects a stage-0 loader to load a Linux kernel and the bootup u-root
initramfs. Once Linux starts, bootup takes over target selection, verification,
staging, and kexec handoff.

The initramfs build keeps bootup's runtime payload to a single u-root
busybox-style binary. TLS roots are compiled into that binary through
`github.com/breml/rootcerts`; distro archive keyrings are not packaged by
default and must be supplied explicitly through provider runtime configuration
or reusable verification hooks.
For downloaded release artifact names, checksums, manifests, and stage-0 usage
examples, see `docs/release.md`.

## Static provider catalog

Bootup embeds a default static catalog of concrete provider targets. The current
default catalog lists Debian bullseye, bookworm, trixie, and forky amd64
netboot, Fedora Server 43 and 44 amd64 netboot, openSUSE Leap, Arch Linux,
GParted Live, local disk boot, plus Ubuntu 24.04.4, 25.10, and 26.04 amd64
netboot.

Use `--catalog` to replace that embedded catalog with a local JSON file:

```sh
bootup --catalog=/etc/bootup/catalog.json --mode=menu
```

The file must be a schema version 1 static catalog:

```json
{
  "schema_version": 1,
  "targets": [
    {
      "id": "debian-trixie-amd64-netboot",
      "provider_id": "debian",
      "name": "Debian trixie amd64 netboot",
      "catalog": {
        "distribution": "debian",
        "release": "trixie",
        "architecture": "amd64",
        "kind": "installer"
      }
    },
    {
      "id": "fedora-44-amd64-server-netboot",
      "provider_id": "fedora",
      "name": "Fedora Server 44 amd64 netboot",
      "catalog": {
        "distribution": "fedora",
        "release": "44",
        "architecture": "amd64",
        "kind": "installer"
      },
      "source": {
        "base_url": "https://download.fedoraproject.org/pub/fedora/linux/releases/44/Server/x86_64/os"
      },
      "lifecycle": {
        "status": "supported",
        "source": "catalog"
      }
    },
    {
      "id": "ubuntu-24044-amd64-netboot",
      "provider_id": "ubuntu",
      "name": "Ubuntu 24.04.4 amd64 netboot",
      "catalog": {
        "distribution": "ubuntu",
        "release": "24.04.4",
        "architecture": "amd64",
        "kind": "installer"
      },
      "source": {
        "base_url": "https://releases.ubuntu.com/24.04",
        "iso_name": "ubuntu-24.04.4-live-server-amd64.iso"
      }
    }
  ]
}
```

The catalog is a replacement, not a merge. If the file contains only the
entries above, bootup exposes only those targets even though other providers may
be compiled into the binary. Invalid catalogs fail startup before provider
target discovery. `source.base_url` is an optional HTTP(S) source root for that
target; `source.iso_name` is an optional pathless installer ISO filename used by
providers such as Ubuntu. Generic Linux catalog targets also use
`source.kernel_path`, optional `source.initrd_path`, and `source.cmdline`.
Targets may also declare `options`; each option is validated catalog data that
can append a kernel command-line fragment when selected with `--option`.
The `localboot` action does not download artifacts and hands off to u-root's
local disk boot path.

The embedded catalog is generated from `internal/catalog/source.json`. Run
`go generate ./internal/catalog` after editing the source.

## Provider runtime config

Use `--provider-config` to point bootup at a JSON file that configures
compiled-in providers before target discovery. Provider entries are keyed by
provider ID, so each distribution source can carry its own operator-selected
trust material without embedding distro keyrings in the default bootup binary:

```json
{
  "providers": {
    "debian": {
      "mirror_url": "https://deb.debian.org/debian",
      "discovery_url": "https://deb.debian.org/debian",
      "discovery_timeout": "5s",
      "keyring_path": "/etc/bootup/trust/debian-archive-keyring.gpg",
      "lifecycle": {
        "trixie": {
          "status": "supported",
          "source": "operator",
          "date": "2028-06-30"
        }
      }
    },
    "ubuntu": {
      "release_url": "https://releases.ubuntu.com/26.04",
      "discovery_url": "https://releases.ubuntu.com/releases",
      "discovery_timeout": "5s",
      "keyring_path": "/etc/bootup/trust/ubuntu-release-keyring.gpg",
      "kernel_sha256": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
      "initrd_sha256": "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
      "lifecycle": {
        "26.04": {
          "status": "supported",
          "source": "operator",
          "date": "2031-05-31"
        }
      }
    },
    "fedora": {
      "release_url": "https://download.fedoraproject.org/pub/fedora/linux/releases/44/Server/x86_64/os",
      "kernel_sha256": "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210",
      "initrd_sha256": "9876543210fedcba9876543210fedcba9876543210fedcba9876543210fedcba"
    }
  }
}
```

Unknown provider IDs, unreadable keyring paths, malformed JSON, invalid release
or discovery URLs, invalid discovery durations, invalid lifecycle entries, and
invalid hash pins fail startup before provider target discovery. Use absolute
paths for keyrings in initramfs and ISO environments. Lifecycle entries are
operator-facing decoration only; they are not artifact trust material.

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
output is redirected. Static targets and discovery-capable provider families
share the first menu; selecting a family runs discovery and opens a second menu
of concrete discovered targets. Force the fallback with `--ui=plain`; use
`--ui=rich` only when a terminal is required and failure is preferable to
fallback.

For non-interactive catalog inspection, list static targets or show one target
without staging artifacts:

```sh
bootup --mode=list-targets
bootup --mode=show-target --target=opensuse-leap-160-amd64-netboot
```

The list output includes each target ID, display name, distribution, release,
architecture, provider, and boot action. The show output includes target
metadata, source artifact references, lifecycle decoration when present, and
declared option definitions.

For a non-interactive discovery diagnostic, list concrete targets for one
compiled-in discovery family:

```sh
bootup --mode=discover-targets --discovery-family=debian --provider-config=/etc/bootup/providers.json
bootup --mode=discover-targets --discovery-family=ubuntu --provider-config=/etc/bootup/providers.json
```

Select catalog-declared target options with repeatable `--option id=value`
flags. Bootup validates the selected option IDs and values before planning. Any
option command-line fragments are appended after provider defaults and before
global `--append-cmdline` text:

```sh
bootup --mode=plan-target \
  --target=opensuse-leap-160-amd64-netboot \
  --option=text-install=true \
  --option=mirror-url=https://mirror.example/opensuse \
  --append-cmdline=console=ttyS1
```

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

For a Debian-capable initramfs, include a local OpenPGP public keyring and a
provider config file in the initramfs:

```sh
cat >/tmp/bootup-providers.json <<'EOF'
{
  "providers": {
    "debian": {
      "keyring_path": "/etc/bootup/trust/debian-archive-keyring.gpg"
    }
  }
}
EOF

scripts/build-initramfs.sh \
  dist/bootup-initramfs.cpio \
  'bootup --mode=menu --prepare-runtime --provider-config=/etc/bootup/providers.json' \
  '' \
  '/tmp/bootup-providers.json:/etc/bootup/providers.json,/usr/share/keyrings/debian-archive-keyring.gpg:/etc/bootup/trust/debian-archive-keyring.gpg'
```

`--prepare-runtime` does not run a user-space DHCP client. Network addressing
should already be provided by the kernel command line, the boot loader, or the
initramfs command used by a local smoke helper. With a purpose-built bootup
kernel, prefer kernel autoconfiguration: build the NIC driver into the kernel,
enable `CONFIG_IP_PNP_DHCP`, and append `ip=::::::dhcp`. DNS servers learned
by the kernel are exposed through `/proc/net/pnp`; bootup copies those hints
into `/etc/resolv.conf` when that file is absent. See `docs/kernel.md` for the
kernel config fragment and validator.

When kernel DHCP is unavailable, bootup can configure a simple static network
before provider operations:

```sh
bootup --mode=menu \
  --net-iface=eth0 \
  --net-address=10.0.2.15/24 \
  --net-gateway=10.0.2.2 \
  --net-dns=10.0.2.3
```

Additional installer or utility parameters can be appended without changing
provider code:

```sh
bootup --mode=boot-target \
  --target=fedora-44-amd64-server-netboot \
  --append-cmdline='inst.vnc console=tty0 console=ttyS1,115200'
```

The helper below performs the same build by generating a temporary provider
config and including the chosen keyring as an initramfs file:

```sh
scripts/build-debian-initramfs.sh /usr/share/keyrings/debian-archive-keyring.gpg
```

To attempt a real QEMU boot into Debian Installer:

```sh
scripts/smoke-real-debian.sh /usr/share/keyrings/debian-archive-keyring.gpg
```

The default provider set also lists Fedora Server 43 and 44 amd64 netboot.
Fedora targets resolve the kernel, initrd, and installer source from the Fedora
Server install tree:

```text
https://download.fedoraproject.org/pub/fedora/linux/releases/44/Server/x86_64/os/images/pxeboot/vmlinuz
https://download.fedoraproject.org/pub/fedora/linux/releases/44/Server/x86_64/os/images/pxeboot/initrd.img
```

Fedora staging uses HTTPS transport trust by default. Custom builds can supply a
Fedora `release_url` override and explicit SHA-256 hashes for the netboot
kernel/initrd if they need pinned artifact verification.

The default provider set also lists Ubuntu 24.04.4, 25.10, and 26.04 amd64
netboot. The 26.04 boot plan uses these official release netboot artifacts by
default:

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

The supported BSD-adjacent target is `mfsbsd-142-amd64`. It uses FreeBSD
`loader.kboot` to boot a verified mfsBSD memory-root payload and reaches the
mfsBSD serial login without attaching target-visible root media. Treat it as a
rescue bridge for BSD workflows, not as stock FreeBSD bootonly installer
support. The target uses `root`/`mfsroot`, enables DHCP and SSH in the mfsBSD
environment, and accepts `--option hostname=<name>` for a non-secret rescue
hostname override. Stock BSD installers, HDT, memdisk ISO images, syslinux
COM32 modules, and iPXE chainload flows still need a dedicated handoff family
rather than the current Linux kernel/initrd kexec path.

Expected local failure modes:

- Missing or unreadable keyring: the helper exits before building.
- No network in the VM: bootup reports route, DNS, TLS, or fetch failures.
- Kernel NIC driver is modular and unavailable: the smoke helper tries to
  include and load the host `e1000` module for QEMU user networking.
- The host kernel used by a generic helper may not provide DNS/route state
  through kernel autoconfiguration, because `CONFIG_IP_PNP` can be unset and
  QEMU NIC drivers can be modules. The mfsBSD product smoke validates kernel
  DHCP support, then passes the expected `10.0.2.3` QEMU resolver into bootup
  before provider downloads.
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
archive trust material and provider config, then pass it to the ISO builder:

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
