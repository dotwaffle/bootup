# bootup

Bootup is a chainloaded Linux/u-root stage-1 environment for dynamic, verified
netboot handoff.

PXE, iPXE, GRUB, or ISO media load a Linux kernel and bootup initramfs. Bootup
then discovers provider targets, verifies downloaded boot artifacts, stages the
selected kernel and initrd, and hands off with kexec.

## Current MVP

- Build-time Go providers.
- Debian bullseye, bookworm, trixie, and forky amd64 netboot targets.
- Fedora Server 43 and 44 amd64 netboot targets.
- Linux-shaped static targets for openSUSE Leap, Arch Linux, and GParted Live.
- Local disk boot through u-root's local boot path.
- Ubuntu 24.04.4, 25.10, and 26.04 amd64 netboot targets.
- Generated embedded static provider catalog with local/hosted replacement and
  opt-in default catalog composition.
- File-backed secret input validation, provider plumbing, and redacted
  diagnostics for targets that declare provider-owned secret consumers.
- Signed local dynamic policy decisions for data-only target, option, and
  secret-reference selection.
- Bright terminal menu with plain serial fallback.
- In-process `kexec_file_load` handoff.
- Embedded Mozilla TLS roots via `github.com/breml/rootcerts`.
- Reusable verification hooks for hashes, SHA256SUMS files, and OpenPGP
  signatures.

Bootup does not commit or package distribution archive keyrings. Callers must
supply trust material to verification hooks when verifying signed distribution
metadata.

See `docs/providers.md` for the static provider catalog, local and hosted
catalog sources, catalog composition, implemented provider discovery, and
policy boundaries.

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
go generate ./internal/catalog
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

Render the current catalog conformance and smoke coverage matrix without
staging artifacts, fetching remote metadata, or contacting mirrors:

```sh
bootup --mode=catalog-matrix
```

The mfsBSD catalog target uses the FreeBSD `loader.kboot` path. It downloads a
pinned mfsBSD ISO and a pinned FreeBSD `base.txz`, extracts the ISO in stage-1,
and hands off with the memory root exposed through `hostfs_root`:

```sh
bootup --mode=plan-target --target=mfsbsd-142-amd64
```

Append installer or utility kernel parameters without editing provider code:

```sh
bootup --mode=boot-target --target=fedora-44-amd64-server-netboot --append-cmdline='inst.vnc console=ttyS1'
```

Provide secret inputs for targets that declare provider-owned secret consumers:

```sh
bootup --mode=stage-target --target=site-installer --secret installer-password=/run/bootup/secrets/installer-password
```

Secret inputs are local file paths only. Bootup validates them before provider
planning, rejects unsafe permissions by default, and redacts source paths and
staged secret paths from diagnostics. The default catalog does not currently
include a distro target that consumes a secret input.

Preview a signed dynamic policy decision without handoff:

```sh
bootup --mode=policy-target --policy-file=/etc/bootup/policy.json --policy-signature=/etc/bootup/policy.json.sig --policy-public-key=/etc/bootup/policy.pub
```

The same policy flags can be used with `plan-target`, `stage-target`, or
`boot-target` to let the authenticated decision supply the target ID, selected
non-secret options, and secret references.

Configure networking directly before provider operations when kernel DHCP is
not available:

```sh
bootup --mode=menu --net-iface=eth0 --net-address=192.0.2.10/24 --net-gateway=192.0.2.1 --net-dns=192.0.2.53
```

List dynamically discovered amd64 netboot targets from a configured provider
source without staging artifacts:

```sh
bootup --mode=discover-targets --discovery-family=debian --provider-config=/etc/bootup/providers.json
```

Use `--discovery-family=ubuntu` or `--discovery-family=fedora` to run the
matching release-index discovery path.

Fedora and Ubuntu netboot targets can be staged from their official HTTPS
release URLs by default. Operators can additionally provide Fedora release URL
overrides and pinned netboot artifact hashes through `--provider-config`.
Ubuntu can also consume release key material plus pinned hashes.

Run the serial selection flow in a Debian-capable build:

```sh
bootup --mode=menu --prepare-runtime
```

Menu mode defaults to `--ui=auto`: it uses the rich keyboard-driven terminal
interface when stdin/stdout are terminals. In the u-root initramfs, auto mode
can reopen `/dev/console` for the rich UI if the init command starts with
non-terminal stdio. It falls back to the plain `target> ` prompt for redirected
input or automation. Discovery-capable providers appear as family entries; when
selected, bootup discovers concrete targets and prompts again. Use `--ui=plain`
to force the fallback or `--ui=rich` to require the rich interface.

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
