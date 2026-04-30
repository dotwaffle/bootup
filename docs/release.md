# Release Artifacts

Bootup releases publish an amd64 artifact set for the default provider-neutral
stage-1 environment. The default release initramfs and ISO do not embed
distribution-specific archive keyrings or trust bundles. Providers that need
stronger validation consume operator-supplied trust material through provider or
application configuration.

## Artifact Set

For release version `${version}`, architecture `amd64`, and Linux kernel
version `${kernel_version}`, the public files are:

| Artifact | Purpose |
| --- | --- |
| `bootup-${version}-linux-amd64` | Standalone bootup Linux amd64 binary |
| `bootup-${version}-linux-${kernel_version}-amd64-bzImage` | Stage-1 kernel for iPXE, GRUB, QEMU, and ISO payloads |
| `bootup-${version}-linux-${kernel_version}-amd64.config` | Kernel config used for the stage-1 kernel |
| `bootup-${version}-initramfs-amd64.cpio.zst` | Default zstd-compressed u-root initramfs |
| `bootup-${version}-hybrid-amd64.iso` | Hybrid BIOS/UEFI ISO with the same kernel and initramfs |
| `bootup-${version}-amd64-manifest.json` | Machine-readable artifact manifest |
| `bootup-${version}-amd64-SHA256SUMS` | SHA-256 checksums for the artifacts and manifest |

The manifest records the schema version, release version, git commit,
architecture, kernel version, artifact roles, byte sizes, SHA-256 digests, and
trust-material posture.

## Verification

Verify downloaded files before booting them:

```sh
sha256sum --check bootup-${version}-amd64-SHA256SUMS
jq '.trustMaterial, .artifacts[] | {role, name, bytes, sha256}' \
  bootup-${version}-amd64-manifest.json
```

`trustMaterial.distributionSpecificEmbedded` is `false` for default release
artifacts.

## iPXE

Serve the kernel and initramfs from HTTP(S) storage and point iPXE at those
files:

```text
kernel http://boot.example/bootup/bootup-${version}-linux-${kernel_version}-amd64-bzImage ip=::::::dhcp console=ttyS0 panic=30
initrd http://boot.example/bootup/bootup-${version}-initramfs-amd64.cpio.zst
boot
```

## GRUB

Copy the kernel and initramfs to the boot volume and add a menu entry:

```text
menuentry "bootup ${version}" {
    linux /bootup/bootup-${version}-linux-${kernel_version}-amd64-bzImage ip=::::::dhcp console=ttyS0 panic=30
    initrd /bootup/bootup-${version}-initramfs-amd64.cpio.zst
}
```

## ISO

Use `bootup-${version}-hybrid-amd64.iso` for virtual media, optical media, or a
USB stick. The ISO boots through GRUB in BIOS and x86_64 UEFI mode and contains
the same kernel and initramfs payloads used by the iPXE and GRUB examples.

On Ubuntu systems with `grub-imageboot`, place the ISO under `/boot/images/`
and run `update-grub` to let the package add a menu entry.

## Local Rehearsal

The release builder writes artifacts to `dist/release/`:

```sh
BOOTUP_RELEASE_VERSION=dev-local scripts/build-release.sh
scripts/check-release-artifacts.sh dist/release
```

Run the BIOS ISO smoke gate used by the release workflow:

```sh
iso="$(find dist/release -maxdepth 1 -type f -name 'bootup-*-hybrid-amd64.iso' | sort | tail -n 1)"
scripts/smoke-iso-bios.sh "${iso}"
```

Useful local overrides:

```sh
BOOTUP_KERNEL_VERSION=6.12.74 BOOTUP_RELEASE_VERSION=dev-local scripts/build-release.sh
BOOTUP_RELEASE_REBUILD_KERNEL=1 BOOTUP_RELEASE_VERSION=dev-local scripts/build-release.sh
```

The release builder reuses the latest `dist/kernel/linux-*-bootup-amd64-bzImage`
when present unless `BOOTUP_RELEASE_REBUILD_KERNEL=1` is set.

## Dependencies

Release builds need Go, Docker, `jq`, `zstd`, `sha256sum`, `grub-mkrescue`,
GRUB BIOS and x86_64 EFI modules, `xorriso`, and network access for Go modules
and kernel sources when a kernel must be built. The BIOS smoke gate also needs
`qemu-system-x86_64` and `timeout`.
