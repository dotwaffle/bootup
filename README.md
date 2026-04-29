# bootup

Bootup is a chainloaded Linux/u-root stage-1 environment for dynamic, verified
netboot handoff.

PXE, iPXE, GRUB, or ISO media load a Linux kernel and bootup initramfs. Bootup
then discovers provider targets, verifies downloaded boot artifacts, stages the
selected kernel and initrd, and hands off with kexec.

## Current MVP

- Build-time Go providers.
- Debian trixie amd64 netboot target.
- Serial-friendly text interface.
- In-process `kexec_file_load` handoff.
- Embedded Mozilla TLS roots via `github.com/breml/rootcerts`.
- Reusable verification hooks for hashes, SHA256SUMS files, and OpenPGP
  signatures.

Bootup does not commit or package Debian archive keyrings. Callers must supply
trust material to verification hooks when verifying signed distribution
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

Stage a target non-interactively. Providers that require distribution trust
material must be configured by the application code that compiles them in:

```sh
bootup --mode=stage-target --target=debian-trixie-amd64-netboot
```
