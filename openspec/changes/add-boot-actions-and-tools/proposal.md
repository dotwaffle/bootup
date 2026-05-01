## Why

Bootup's current provider model assumes every target is a Linux installer with
both a kernel and initrd. The next catalog expansion needs local disk boot,
Linux utility targets, installer command-line options, and explicit deferral of
BSD/memdisk targets that do not fit the current handoff path.

## What Changes

- Add a typed boot action to targets and boot plans, with `linux-kexec` as the
  default and `localboot` for local disk handoff.
- Add a compiled local boot provider that invokes the u-root local disk boot
  path without downloading artifacts.
- Add a generic compiled Linux boot provider for static kernel/initrd targets
  described by catalog source metadata.
- Add default catalog entries for openSUSE, Arch Linux, GParted Live, and
  MemTest86+ targets that use the Linux/kexec path.
- Allow operators to append kernel command-line parameters to selected targets
  for installer options such as serial console, VNC, rescue, and automation
  flags.
- Allow operators to explicitly configure interface address, default route, and
  DNS before network validation and artifact retrieval.
- Document BSD, HDT, memdisk ISO, and chainload targets as deferred until a
  dedicated executor family exists.

## Capabilities

### New Capabilities
- `bootup-boot-actions`: Typed boot actions for Linux kexec, local disk boot,
  and deferred future executor families.

### Modified Capabilities
- `bootup-netboot`: Boot planning, staging, and handoff behavior now dispatches
  by action and supports optional initrd artifacts.
- `bootup-static-catalog-source`: The embedded catalog now includes local boot,
  generic Linux static targets, and per-target source metadata for kernel,
  initrd, and command line fields.
- `bootup-static-provider-catalog`: Target metadata validation now accepts an
  optional boot action and supports catalog distributions that differ from the
  compiled provider responsible for planning.
- `bootup-provider-runtime-config`: Runtime startup configuration now includes
  global command-line append and explicit network configuration.

## Impact

- Affected code: provider data model, catalog generation and validation,
  app planning/staging flow, handoff executor, initramfs applet list, runtime
  network preparation, CLI flags, docs, and provider registration.
- APIs: `provider.Target` and `provider.BootPlan` gain boot action metadata;
  `provider.SourceEntry` gains static Linux source fields.
- Dependencies: no new third-party dependency is required. The initramfs adds
  the existing u-root `boot` applet so local disk boot can use u-root's parser
  and kexec path.
