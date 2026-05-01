## Context

Bootup currently selects a provider target, asks the provider for a Linux
kernel/initrd plan, stages both artifacts, then executes `kexec_file_load`.
That works for Debian, Ubuntu, and Fedora netboot installers. It does not
represent local disk boot, targets without an initrd, or future boot methods
such as memdisk ISO and chainloading.

The spike showed that the near-term supported set is Linux-shaped: openSUSE,
Arch, GParted, and MemTest86+ can be represented as kernel/initrd or kernel-only
targets. BSD ISO/memstick paths and HDT depend on memdisk/syslinux/chainload
semantics and should remain out of the executable catalog until a separate
executor exists.

## Goals / Non-Goals

**Goals:**
- Preserve existing Debian, Ubuntu, and Fedora behavior.
- Add a typed action model without forcing every existing provider to change
  its public planning logic.
- Support local disk boot through u-root's existing boot command.
- Add a generic static Linux provider for catalog-defined kernel/initrd targets.
- Allow optional initrd for kernel-only utility targets such as MemTest86+.
- Add explicit operator knobs for appended kernel parameters and network setup.

**Non-Goals:**
- Do not implement BSD, memdisk ISO, syslinux COM32, or iPXE chainload
  execution.
- Do not add distro-specific keyrings or detached-signature policy.
- Do not add URL-hosted catalogs or executable runtime scripts.
- Do not attempt full installer automation beyond command-line parameter
  plumbing.

## Decisions

1. Use `provider.BootAction` on both `Target` and `BootPlan`.
   `linux-kexec` remains the default when the field is empty, preserving old
   catalog and provider behavior. `localboot` is the first non-Linux-artifact
   action. Future values can describe `memdisk`, `chainload`, or `multiboot`
   without overloading catalog kind.

2. Keep planning through providers.
   Local disk boot is exposed by a small compiled provider that returns a
   `localboot` plan and no artifacts. This keeps selection, planning, and error
   reporting uniform instead of adding a special menu branch.

3. Add a generic `linux` provider for static kernel/initrd targets.
   The provider reads target source metadata for `base_url`, `kernel_path`,
   optional `initrd_path`, and `cmdline`. This avoids one bespoke package for
   every Linux-shaped distro or utility while keeping provider code compiled
   into the binary.

4. Dispatch handoff by action.
   `linux-kexec` continues to use the in-process syscall loader. `localboot`
   executes the u-root `boot` applet, passing appended command-line parameters
   when present. Unsupported actions fail closed with a clear error.

5. Treat command-line append as an app-level operator option.
   Providers still produce their defaults. The app appends operator parameters
   after planning and before staging, so the feature applies consistently to
   all Linux-shaped targets and the u-root local boot command.

6. Configure networking before validation.
   `NetworkPreparer` gains optional interface, address, gateway, and DNS
   fields. When set, it runs `ip` commands and writes resolver configuration
   before checking for a configured non-loopback interface.

## Risks / Trade-offs

- Generic Linux catalog entries can point to stale upstream paths. Mitigation:
  keep entries HTTPS-only by default, test planning hermetically, and leave
  live smoke tests explicit.
- Local disk boot depends on u-root's `boot` applet and the local disk's boot
  configuration. Mitigation: include the applet in the initramfs and report
  command failure without hiding diagnostics.
- Appended kernel parameters can conflict with provider defaults. Mitigation:
  append only when explicitly requested and document that the operator controls
  conflict resolution.
- BSD support is deferred even though some FreeBSD LinuxBoot work exists.
  Mitigation: document the reason and avoid presenting unsupported entries as
  bootable.

## Migration Plan

Existing catalog documents remain valid because omitted actions default to
`linux-kexec` and omitted static Linux source fields are only required for the
new generic provider. Existing CLI modes and provider config files continue to
load.
