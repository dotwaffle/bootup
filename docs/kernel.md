# Bootup Kernel Configuration

Bootup can run on a normal distro kernel for local testing, but the preferred
deployment shape is a purpose-built kernel paired with the bootup initramfs.
That kernel should bring up networking before `/init` by using kernel IP
autoconfiguration:

```text
console=tty0 console=ttyS0,115200n8 panic=30 ip=::::::dhcp
```

Kernel DHCP runs before initramfs modules can be loaded, so the boot NIC driver
must be built in. For the amd64/QEMU and common bare-metal target, start from a
normal x86_64 configuration and merge:

```sh
configs/kernel/bootup-amd64-qemu.fragment
```

Or build the bootup kernel through the repository's Docker builder:

```sh
scripts/build-kernel.sh
```

The script reads kernel.org release metadata, builds the latest stable upstream
Linux release inside Docker, copies
`configs/kernel/bootup-amd64-qemu.fragment` to `.config`, runs
`make olddefconfig`, validates the resolved config, and writes both the kernel
and resolved config under `dist/kernel/`. It requires `curl`, `jq`, and Docker.
Override the Linux version with `BOOTUP_KERNEL_VERSION` for pinned builds, for
example:

```sh
BOOTUP_KERNEL_VERSION=6.12.74 scripts/build-kernel.sh
```

The fragment is intentionally biased toward early-boot reliability rather than
a general distro kernel. It requires:

- zstd kernel image compression with `CONFIG_KERNEL_ZSTD`.
- zstd initramfs decompression with `CONFIG_RD_ZSTD`.
- Built-in kexec support.
- Built-in serial and VGA/framebuffer console support.
- Built-in IPv4 DHCP autoconfiguration plus `e1000` and `virtio_net`.
- Built-in keyboard, USB HID, virtio block, NVMe, SATA, USB storage, ext4, and
  VFAT support so bootup can inspect common local media for chainload inputs.

`CONFIG_CMDLINE_BOOL` supplies fallback defaults:

```text
console=tty0 console=ttyS0,115200n8 earlyprintk=ttyS0,115200 panic=30 ip=::::::dhcp
```

Do not set `CONFIG_CMDLINE_OVERRIDE` for normal bootup builds; PXE, iPXE, GRUB,
or an ISO bootloader should still be able to provide deployment-specific
arguments. For modern kernels, the fragment uses the smaller fbdev
`CONFIG_FB_SIMPLE` path and explicitly disables `CONFIG_DRM_SIMPLEDRM` because
those options conflict.

Validate a candidate kernel config with:

```sh
scripts/check-kernel-config.sh /path/to/.config
```

For example, the current host kernel can be inspected with:

```sh
scripts/check-kernel-config.sh /boot/config-$(uname -r)
```

It is acceptable for that host-kernel check to fail on developer machines. The
real Debian QEMU smoke keeps a static-network fallback for that case. Release
kernels intended for `ip=::::::dhcp` should pass the checker.
