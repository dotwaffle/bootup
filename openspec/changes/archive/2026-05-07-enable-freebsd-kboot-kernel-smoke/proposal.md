## Why

The FreeBSD kboot spike proved that `loader.kboot` can run from bootup and see
a FreeBSD block-device payload, but the current bootup kernel does not expose
the Linux metadata interfaces the loader needs to recover `boot_params` and EFI
memory-map state. We need a focused kernel-prerequisite smoke before designing
any production FreeBSD boot action.

## What Changes

- Extend the bootup amd64 kernel fragment and validator with the
  `loader.kboot` metadata prerequisites.
- Add an explicit smoke procedure for building a temporary initramfs/ISO with
  `loader.kboot`, presenting the FreeBSD installer ISO as a block device, and
  checking whether the handoff gets past the prior metadata blocker.
- Keep FreeBSD and mfsBSD targets out of the executable default catalog until
  that smoke reaches a FreeBSD installer or mfsBSD shell.
- Do not commit `loader.kboot`, FreeBSD release payloads, generated kernels,
  initramfs images, ISOs, or VM disks.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `bootup-kernel`: add FreeBSD kboot metadata prerequisites to the documented
  and locally validated bootup kernel configuration.
- `bootup-freebsd-kboot-handoff`: add the required kernel-prerequisite smoke
  evidence before the handoff can be called viable.

## Impact

- Kernel config fragment and kernel config validation helper.
- Kernel config validation fixtures/tests.
- A new or updated smoke script/documentation for the FreeBSD kboot proof path.
- No provider catalog entries and no committed binary payloads.
