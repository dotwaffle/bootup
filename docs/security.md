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

Provider runtime configuration can also set discovery source URLs, local
discovery metadata paths, discovery timeouts, and lifecycle decoration maps.
Those fields only control where compiled-in discovery code looks and what
informational lifecycle text is shown to operators; they do not supply or
replace artifact authenticity checks.

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

Signed dynamic policy decisions are authenticated before parsing. Bootup
requires a detached Ed25519 signature and public key for `--policy-file` or
`--policy-url`; the signature covers the raw policy document bytes. HTTPS is
required for remote policy URLs, but transport security does not replace local
signature trust. A valid policy can only select an existing target, validated
non-secret option values, and secret references handled by the secret input
path. It cannot define executable behavior, providers, trust roots, artifact
pins, or new command-line fragments.

Policy freshness is fail-closed. A decision must carry `expires_at`, or must
carry `published_at` when `--policy-max-age` is used. Expired decisions,
signature failures, malformed JSON, unsupported targets/options/secrets, and
missing required secrets fail before provider planning. Policy diagnostics
record posture and decision IDs, not response bodies, trust bytes, secret
values, or policy source paths.

The Debian provider fails closed unless Debian archive trust material is
provided through configuration. The embedded Fedora catalog carries SHA-256
pins from Fedora `.treeinfo` metadata for the Server pxeboot kernel and initrd.
Custom or discovered Fedora targets without catalog pins fetch `.treeinfo` over
HTTPS during planning and fail closed unless both pxeboot SHA-256 checksums are
present; explicit Fedora runtime hash pins override both catalog pins and
metadata lookup. The Ubuntu provider can stage the official 26.04 netboot
kernel and initrd over HTTPS by default; callers that need a stronger chain can
configure Ubuntu release key material plus pinned SHA-256 hashes for those
netboot artifacts. Ubuntu's signed release `SHA256SUMS` currently covers the
ISO set, not the extracted netboot kernel and initrd.

## Secret inputs

Target options, provider runtime configuration, catalogs, boot plans, loader
arguments, kernel command lines, logs, and diagnostics are treated as
operator-visible surfaces. Do not place passwords, password hashes, SSH keys,
API tokens, or other secret material in those fields.

Targets that need sensitive material must declare separate secret inputs.
Operators provide each value as a local file-backed input with repeatable
`--secret id=/absolute/path` flags. Bootup validates the path before provider
planning, requires a regular readable file below the configured size limit, and
rejects group- or other-readable files by default. Secret values are kept out
of public boot plan fields; providers receive a secret store and can request a
private staged file when the target declaration uses `staged-file` delivery.

Diagnostics may include secret IDs so an operator can see which declaration was
involved, but they must not include the secret value, the operator source path,
the staged private path, value hashes, provider config contents, or derived boot
arguments containing secret material.

Local builders that want a self-contained Debian-capable initramfs can include
their chosen Debian archive public keyring as an initramfs file and point
`--provider-config` at a JSON file that references that path. The trust root is
still an explicit operator input, not repository content or a default release
payload.
