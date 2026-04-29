# Bootup Kernel Configuration

Bootup can run on a normal distro kernel for local testing, but the preferred
deployment shape is a purpose-built kernel paired with the bootup initramfs.
That kernel should bring up networking before `/init` by using kernel IP
autoconfiguration:

```text
ip=::::::dhcp console=ttyS0 panic=30
```

Kernel DHCP runs before initramfs modules can be loaded, so the boot NIC driver
must be built in. For the initial amd64/QEMU target, start from a normal x86_64
configuration and merge:

```sh
configs/kernel/bootup-amd64-qemu.fragment
```

The fragment is intentionally small. It requires initramfs support, kexec,
zstd initramfs decompression, IPv4 DHCP autoconfiguration, and built-in `e1000`
and `virtio_net` drivers.

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
