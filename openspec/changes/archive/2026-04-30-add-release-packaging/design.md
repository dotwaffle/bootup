## Context

Bootup currently has local helpers for building the stage-1 binary inside a
u-root initramfs, building a purpose-built amd64 kernel, and wrapping both into
a hybrid BIOS/UEFI ISO. CI validates the Go code, script syntax, fixture builds,
and VM test compilation, but it does not produce a stable release bundle or
publish artifacts.

The public release surface should be smaller and more stable than the working
`dist/` tree. Generated intermediates can remain implementation details, while
published files need predictable names, checksums, metadata, validation, and
operator documentation.

Distribution trust roots are intentionally outside the default release payload.
Release packaging should not add Debian-specific keyring behavior or any other
distribution carve-out. Providers that require stronger validation continue to
consume operator-supplied trust material through provider/application
configuration.

## Goals / Non-Goals

**Goals:**
- Publish a repeatable amd64 release bundle containing the bootup binary,
  kernel image/config, zstd initramfs, hybrid ISO, checksum file, and manifest.
- Make the release artifact names stable enough for operators, automation, and
  documentation.
- Validate artifacts before publication, including manifest/checksum integrity
  and at least one ISO boot smoke.
- Document which artifacts are intended for iPXE, GRUB, and ISO media.
- Ensure default release artifacts do not embed distribution-specific trust
  bundles.

**Non-Goals:**
- Designing a global provider trust policy or embedding distribution keyrings in
  official artifacts.
- Publishing provider-specific initramfs or ISO variants.
- Solving Secure Boot signing or measured boot.
- Adding non-amd64 release targets.
- Replacing the local development scripts with a separate packaging system.

## Decisions

1. **Use a release assembly script over direct CI-only steps.**

   Add a repository script that builds or collects the existing kernel,
   initramfs, binary, and ISO outputs into `dist/release/`. CI should call the
   same script operators can run locally. The alternative was to encode assembly
   entirely in GitHub Actions, but that would make local release rehearsals and
   debugging harder.

2. **Publish a small, versioned artifact set.**

   Release files should use a bootup release version and architecture in their
   names. Kernel artifacts should also include the Linux kernel version because
   the kernel image is operationally distinct from the bootup release version.
   The raw initramfs can remain a build intermediate; the published initramfs is
   the zstd-compressed image used by current launch paths.

3. **Generate both `SHA256SUMS` and a machine-readable manifest.**

   `SHA256SUMS` supports simple operator verification with common tools. The
   JSON manifest gives automation artifact roles, sizes, checksums, build
   metadata, and the explicit trust-material posture without parsing filenames.
   The alternative was checksums only, but that leaves release consumers without
   a stable role-to-file mapping.

4. **Keep release artifacts provider-neutral.**

   The default release initramfs and ISO use the normal provider set and do not
   embed distribution-specific trust bundles. Provider verification remains an
   operator configuration concern. The alternative was to publish Debian-capable
   artifacts as a convenience, but that would make one provider's trust root a
   special case in a project that is expected to support many image sources.

5. **Separate normal CI from release publication.**

   Existing CI should continue to run on pushes and pull requests. Release
   publication should live in a tag/workflow-dispatch path with permissions to
   create or update GitHub release assets. This keeps broad release permissions
   out of ordinary validation jobs.

## Risks / Trade-offs

- Release builds depend on Docker, kernel.org metadata, GRUB, xorriso, zstd, and
  QEMU availability -> document dependencies and fail early with clear messages.
- Building the latest stable kernel during release can make tag reruns produce a
  different kernel -> record the kernel version in artifact names and manifest,
  and allow `BOOTUP_KERNEL_VERSION` to pin the build.
- QEMU/ISO smoke can be slow or unavailable in some CI environments -> make it a
  release gate first, while keeping more expensive UEFI smoke available as a
  local/manual validation path.
- The manifest schema may need to grow for more architectures or signatures ->
  include a manifest schema version and avoid encoding release semantics solely
  in filenames.
- Operators may assume official artifacts include distribution keyrings -> state
  explicitly in docs and manifest that default artifacts embed no
  distribution-specific trust bundles.
