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
trust-root selection outside the shipped binary until a concrete distribution
trust policy is designed.

Keyrings are automatically detected when they are OpenPGP public key material:
ASCII-armored exported public keys or binary OpenPGP keyrings such as Debian
archive keyring files. GnuPG keybox databases, trust databases, and unrelated
PEM files are not accepted as keyrings.
