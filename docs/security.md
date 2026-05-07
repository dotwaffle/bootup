# Secure Boot and lockdown notes

Bootup's current MVP handoff path uses `kexec_file_load` followed by
`reboot(LINUX_REBOOT_CMD_KEXEC)`.

Known failure modes:

- Kernel lockdown can reject kexec loading when Secure Boot is enabled.
- The running kernel can be built without kexec support.
- The platform can require signed target kernels for `kexec_file_load`.
- LSM or capability policy can deny the syscall even when the kernel supports
  it.

Bootup must treat these as handoff failures, report the syscall error on the
serial interface, and keep the current environment alive for diagnosis. It must
not discard logs or reboot into an unverified fallback.

Future Secure Boot support should decide whether bootup only accepts distro
signed kernels, project-signed kernels, or a machine-owner-key flow.

## Trust material

Bootup embeds Mozilla TLS roots with `github.com/breml/rootcerts` so HTTPS can
work without packaging a CA bundle file in the initramfs.

Distribution archive keyrings are not committed to this repository and are not
packaged into the default initramfs. The public `verify` package accepts
caller-supplied readers for artifacts, hashes, signatures, and keyrings, plus
file helpers for callers that stage trust material on disk. This keeps
trust-root selection outside the shipped binary and lets operators choose the
trust material for each configured provider source.

Keyrings are automatically detected when they are OpenPGP public key material:
ASCII-armored exported public keys or binary OpenPGP keyrings such as Debian
archive keyring files. GnuPG keybox databases, trust databases, and unrelated
PEM files are not accepted as keyrings.

Providers that need distribution trust material receive it through explicit
runtime or application-level configuration. The default binary accepts a
provider runtime configuration file with `--provider-config`, but it does not
commit, generate, package, or embed a distribution keyring by default.

Provider runtime configuration can also set discovery source URLs, discovery
timeouts, and lifecycle decoration maps. Those fields only control where
compiled-in discovery code looks and what informational lifecycle text is shown
to operators; they do not supply or replace artifact authenticity checks.

Dynamic provider discovery returns candidate targets and optional lifecycle
decoration as provider data. Lifecycle fields such as `supported`, `obsolete`,
`eol`, or `unknown` are not trust material and are not used as signatures,
checksums, keyrings, transport policy, or authenticity signals for downloaded
boot artifacts.

Hosted static catalogs are authenticated before parsing. Operators must pin the
raw catalog bytes with `--catalog-sha256` or provide a detached Ed25519
signature and public key with `--catalog-signature` and
`--catalog-public-key`. Catalog signatures and digest pins only authenticate the
catalog document itself; they do not replace distribution archive keyrings,
provider artifact hashes, or provider-owned signature verification. Hosted
catalog freshness metadata is also a catalog acceptance policy, not an artifact
authenticity signal.

The Debian provider fails closed unless Debian archive trust material is
provided through configuration. The Fedora provider can stage Fedora Server
netboot kernel and initrd artifacts over HTTPS by default; callers that need
pinned verification can configure a Fedora release URL plus explicit SHA-256
hashes for those netboot artifacts. The Ubuntu provider can stage the official
26.04 netboot kernel and initrd over HTTPS by default; callers that need a
stronger chain can configure Ubuntu release key material plus pinned SHA-256
hashes for those netboot artifacts. Ubuntu's signed release `SHA256SUMS`
currently covers the ISO set, not the extracted netboot kernel and initrd.

Local builders that want a self-contained Debian-capable initramfs can include
their chosen Debian archive public keyring as an initramfs file and point
`--provider-config` at a JSON file that references that path. The trust root is
still an explicit operator input, not repository content or a default release
payload.
