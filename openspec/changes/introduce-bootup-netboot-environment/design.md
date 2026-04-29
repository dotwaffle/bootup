## Context

Bootup is intended to run after firmware or an existing bootloader has done the
minimum work required to load a Linux kernel and initramfs. PXE, iPXE, GRUB,
and ISO media remain useful as stage-0 delivery mechanisms, but bootup takes
over once Linux is running so it can use Go libraries, real networking, TLS,
signature verification, richer UI backends, and kexec.

The initial target is Debian trixie amd64 netboot because Debian publishes
stable archive metadata, signed release files, installer checksums, and direct
kernel/initrd artifacts in predictable archive paths. Ubuntu 26.04 remains a
good follow-on target, but its current release checksum listing does not cover
the netboot tarball in the same direct way, so it is not the verification-first
MVP.

## Goals / Non-Goals

**Goals:**

- Build a chainloadable u-root initramfs containing bootup and the minimal
  runtime tools needed for network boot orchestration.
- Provide a build-time provider model where Go provider modules are compiled
  into the distributed image.
- Implement a Debian trixie amd64 provider that resolves, verifies, downloads,
  stages, and kexecs the Debian Installer netboot kernel and initrd.
- Keep bootup's runtime payload to a single binary where practical; TLS roots
  are compiled into the binary and distro archive keyrings are not committed or
  packaged by default. Debian archive trust material is supplied by application
  builds that intentionally compile it into their own binary.
- Make the plain serial interface first-class for IPMI, KVM, and console-only
  workflows.
- Keep the runtime architecture ready for framebuffer UI without requiring it
  for the MVP.
- Exercise the boot path under QEMU/vmtest.

**Non-Goals:**

- Runtime plugin loading.
- A complete netboot.xyz replacement catalog in the first change.
- Secure Boot support beyond documenting detected failure modes.
- A framebuffer game or animated UI in the first implementation.
- Automated disk provisioning or unattended installation policy.

## Decisions

### Use stage-0 bootloaders only as delivery mechanisms

Bootup will be packaged so PXE, iPXE, GRUB, and ISO flows can load a Linux
kernel and u-root initramfs. Once bootup starts, operating system selection,
download, verification, and handoff happen inside Linux.

Alternatives considered:
- Extend iPXE scripting: lower boot payload cost, but keeps the project inside
  a limited scripting and UI environment.
- Build a standalone firmware payload: more control, but much higher hardware
  and platform complexity before the core idea is proven.

### Compile providers at build time

Providers will be ordinary Go packages registered into bootup at build time.
Each provider exposes discovery, planning, verification, and boot preparation
behavior. This follows the u-root/gobusybox style and keeps images
reproducible and auditable.

Alternatives considered:
- Runtime Go plugins: not suitable for this environment and awkward across
  architectures.
- Downloaded scripts: flexible, but weakens auditability and makes early boot
  trust harder to reason about.

### Start with Debian trixie amd64

The Debian provider will resolve the amd64 installer artifacts from the trixie
archive, verify signed archive metadata with caller-supplied Debian archive
trust material, verify checksums for the selected installer files, then stage
the kernel and initrd for kexec.

Alternatives considered:
- Ubuntu 26.04: current and useful, but netboot artifact verification is less
  clean for an MVP.
- Alpine: simple and small, but Debian better exercises archive metadata
  verification and installer handoff complexity.

### Make serial UI the baseline

The MVP UI will render a plain menu that works over serial consoles and
console-like KVM/IPMI sessions. Framebuffer rendering is a backend direction,
not a dependency of the first reliable boot path.

Alternatives considered:
- Framebuffer-first UI: better demo value, but it risks delaying the core boot
  and verification work.
- No UI, command-line only: simpler to test, but misses the central operator
  workflow of interactive target selection.

### Treat verification as part of planning, not a post-download detail

Provider boot plans include the metadata, checksums, and artifacts required for
verification. Provider configuration supplies the trust material. Bootup MUST
fail closed if it cannot establish a trusted path from explicit trust material
to downloaded artifacts.

Alternatives considered:
- HTTPS-only trust: protects transport but does not provide distro release
  authenticity or mirror integrity.
- Optional verification: easier initial implementation, but undermines the
  value of a boot environment that downloads kernels and initrds.

### Prefer binary-embedded TLS roots over CA bundle files

Bootup imports `github.com/breml/rootcerts` from the main package so Go can use
compiled-in Mozilla TLS roots when the system certificate pool is missing. This
keeps the initramfs from needing a separate CA bundle data file while preserving
normal `crypto/x509` behavior.

Alternatives considered:
- Package `/etc/ssl/certs/ca-certificates.crt`: simple, but violates the
  single-binary runtime direction.
- Disable TLS verification: unacceptable for a network boot tool.

## Risks / Trade-offs

- Secure Boot or kernel lockdown blocks kexec -> detect the failure, report a
  clear error, and document the unsupported path until signed-kernel support is
  designed.
- Early boot time is wrong -> include a time sanity step and support NTP/TLS
  time correction before TLS-heavy flows.
- Network setup varies by hardware -> keep DHCP/DNS behavior explicit and
  expose failures in the serial UI and logs.
- Debian archive layout changes -> keep URLs inside the provider and cover the
  provider with integration tests that fail when expected artifacts move.
- Distro archive trust roots expire or rotate -> require explicit trust
  material input and test verification against supplied keyrings.
- kexec behavior differs by architecture -> scope the first provider to amd64
  and add architectures only after the amd64 path is stable.

## Migration Plan

This is a new capability, so there is no existing user migration. The first
implementation should add bootup as a new buildable image and keep all launch
paths opt-in.

Rollback is to boot the previous PXE/iPXE/GRUB target directly instead of
chainloading bootup.

## Open Questions

- Which exact Linux kernel configuration should be used for the bootup stage-1
  image?
- What is the minimum useful persistent logging story for failed remote boots?
