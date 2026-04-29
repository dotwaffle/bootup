# bootup

Bootup is a chainloaded Linux/u-root stage-1 environment for dynamic, verified
netboot handoff.

PXE, iPXE, GRUB, or ISO media load a Linux kernel and bootup initramfs. Bootup
then discovers provider targets, verifies downloaded boot artifacts, stages the
selected kernel and initrd, and hands off with kexec.

## Current MVP

- Build-time Go providers.
- Debian trixie amd64 netboot target.
- Ubuntu 26.04 amd64 netboot target.
- Serial-friendly text interface.
- In-process `kexec_file_load` handoff.
- Embedded Mozilla TLS roots via `github.com/breml/rootcerts`.
- Reusable verification hooks for hashes, SHA256SUMS files, and OpenPGP
  signatures.

Bootup does not commit or package distribution archive keyrings. Callers must
supply trust material to verification hooks when verifying signed distribution
metadata.

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
requirements must be configured by the application code that compiles them in:

```sh
bootup --mode=stage-target --target=debian-trixie-amd64-netboot
```

Ubuntu 26.04 netboot can be staged from the official HTTPS release URLs by
default. Custom builds can additionally provide Ubuntu release key material and
pinned netboot artifact hashes.

Run the serial selection flow in a Debian-capable build:

```sh
bootup --mode=menu --prepare-runtime
```

Build a local single-binary Debian-capable image by generating ignored Go
source from an OpenPGP public keyring, then building normally:

```sh
go run ./cmd/bootup-keyring-source -o internal/trustmaterial/debian_archive_keyring_generated.go /usr/share/keyrings/debian-archive-keyring.gpg
go build -trimpath -o /tmp/bootup ./cmd/bootup
```

Build a Debian-capable initramfs without leaving generated keyring source in the
worktree:

```sh
scripts/build-debian-initramfs.sh /usr/share/keyrings/debian-archive-keyring.gpg
```

Attempt a real QEMU smoke boot that stages live Debian Installer artifacts and
loads them through kexec:

```sh
scripts/smoke-real-debian.sh /usr/share/keyrings/debian-archive-keyring.gpg
```
