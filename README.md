# bootup

Bootup is a chainloaded Linux/u-root stage-1 environment for dynamic, verified
netboot handoff.

PXE, iPXE, GRUB, or ISO media load a Linux kernel and bootup initramfs. Bootup
then discovers provider targets, verifies downloaded boot artifacts, stages the
selected kernel and initrd, and hands off with kexec.

## Current MVP

- Build-time Go providers.
- Debian bullseye, bookworm, and trixie amd64 netboot targets.
- Ubuntu 24.04.4, 25.10, and 26.04 amd64 netboot targets.
- Embedded static provider catalog with local JSON replacement.
- Bright terminal menu with plain serial fallback.
- In-process `kexec_file_load` handoff.
- Embedded Mozilla TLS roots via `github.com/breml/rootcerts`.
- Reusable verification hooks for hashes, SHA256SUMS files, and OpenPGP
  signatures.

Bootup does not commit or package distribution archive keyrings. Callers must
supply trust material to verification hooks when verifying signed distribution
metadata.

See `docs/providers.md` for the current static provider catalog model and the
deferred dynamic discovery and policy modes.

```go
err := verify.Artifact(verify.ArtifactInput{
	Artifact:  kernel,
	SHA256Sums: sums,
	Name:      "debian-installer/amd64/linux",
})
```

## Development

Run the normal checks:

```sh
go test ./...
golangci-lint run
go build -trimpath -o /tmp/bootup ./cmd/bootup
```

Run the tagged vmtest package compile/skip path:

```sh
go test -tags vmtest ./test/vmtest
```

Run vmtest with an auto-built cached latest-stable kernel:

```sh
test/vmtest/run
```

Build a purpose-built bootup kernel with Docker:

```sh
scripts/build-kernel.sh
```

The kernel builder follows kernel.org's latest stable release by default. Set
`BOOTUP_KERNEL_VERSION` to pin a specific upstream release.

Build a bootable hybrid ISO from the kernel and a menu-mode initramfs:

```sh
scripts/build-iso.sh
```

Run the ISO under QEMU BIOS:

```sh
scripts/run-qemu-iso.sh
```

Build an initramfs:

```sh
scripts/build-initramfs.sh
```

Run QEMU with a local kernel and the generated initramfs:

```sh
scripts/run-qemu.sh
```

Check whether a kernel config is suitable for kernel-side DHCP with bootup:

```sh
scripts/check-kernel-config.sh /path/to/.config
```

Stage a target non-interactively. Providers with stronger distribution trust
requirements can be configured through a provider runtime config file:

```sh
bootup --mode=stage-target --target=debian-trixie-amd64-netboot --provider-config=/etc/bootup/providers.json
```

Ubuntu netboot targets can be staged from their official HTTPS release URLs by
default. Operators can additionally provide Ubuntu release key material and
pinned netboot artifact hashes through `--provider-config`.

Run the serial selection flow in a Debian-capable build:

```sh
bootup --mode=menu --prepare-runtime
```

Menu mode defaults to `--ui=auto`: it uses the rich keyboard-driven terminal
interface when stdin/stdout are terminals. In the u-root initramfs, auto mode
can reopen `/dev/console` for the rich UI if the init command starts with
non-terminal stdio. It falls back to the plain `target> ` prompt for redirected
input or automation. Use `--ui=plain` to force the fallback or `--ui=rich` to
require the rich interface.

Build a Debian-capable initramfs by including an operator-supplied OpenPGP
public keyring and generated provider config in the initramfs:

```sh
scripts/build-debian-initramfs.sh /usr/share/keyrings/debian-archive-keyring.gpg
```

Attempt a real QEMU smoke boot that stages live Debian Installer artifacts and
loads them through kexec:

```sh
scripts/smoke-real-debian.sh /usr/share/keyrings/debian-archive-keyring.gpg
```

Attempt the matching Ubuntu 26.04 HTTPS netboot smoke:

```sh
scripts/smoke-real-ubuntu.sh
```
