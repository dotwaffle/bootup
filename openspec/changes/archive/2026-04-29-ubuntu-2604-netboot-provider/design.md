## Context

Ubuntu's current 26.04 release page exposes a netboot tarball and extracted
`netboot/amd64/linux` and `netboot/amd64/initrd` files. The same release page
also publishes `SHA256SUMS` and `SHA256SUMS.gpg`, but those sums cover the ISO
and WSL images, not the extracted netboot files. Bootup's provider contract
requires verified boot artifacts before staging and kexec, so the Ubuntu
provider cannot treat the signed ISO checksum file as proof for the netboot
kernel or initrd. For this first Ubuntu slice, HTTPS is accepted as the default
transport trust for the extracted netboot files, with pinned hashes available
for callers that want stronger guarantees.

## Decisions

### Expose the target now

The catalog should include Ubuntu 26.04 so the UI and provider model can grow
beyond Debian. Planning is deterministic and points at official release URLs.

### Use HTTPS by default, with optional pinned hashes

The provider stages the netboot kernel and initrd from official HTTPS release
URLs by default. It also accepts explicit SHA-256 hashes for both artifacts
through build-time configuration. If one hash is supplied, both must be
supplied, so custom builds do not accidentally mix pinned and unpinned boot
artifacts.

### Verify signed release metadata separately

When caller-supplied Ubuntu release key material is present, the provider
verifies `SHA256SUMS` against `SHA256SUMS.gpg` and checks that the referenced
live-server ISO is present in the signed sums. When explicit boot artifact
hashes are present, the boot artifacts are verified against those hashes.

### Keep trust material external

As with Debian, no Ubuntu keyring is committed or embedded by default. Callers
that want a fully staging-capable Ubuntu build can generate or provide trust
material in their own build configuration.

## Risks / Trade-offs

- HTTPS-only netboot staging is weaker than Debian's signed installer checksum
  chain, but it matches the trust model many operators already use for Ubuntu
  ISO downloads.
- Ubuntu may add signed netboot checksums later. The provider shape keeps that
  upgrade local to the staging implementation.
- The provider can be listed by default even though staging fails closed without
  additional trust configuration.

## Rollout

1. Add Ubuntu provider package and register it by default.
2. Add unit coverage for target metadata, plan URLs, HTTPS-only staging, and
   optional verification behavior.
3. Document the current Ubuntu verification trade-off.
4. Run the full test and lint suite.
