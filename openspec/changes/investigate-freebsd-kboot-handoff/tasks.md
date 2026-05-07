## 1. Loader Acquisition

- [x] 1.1 Identify whether `loader.kboot` is available in official FreeBSD 15.0 artifacts or must be built from source
- [x] 1.2 Document the exact source, build command, output path, and SHA-256 hash for the loader used by the spike
- [x] 1.3 Confirm the loader is built or obtained outside tracked repository paths

## 2. Payload Layout

- [x] 2.1 Select a stock FreeBSD installer payload, mfsBSD payload, or both for the first QEMU proof
- [x] 2.2 Document the kernel, module, root filesystem, and configuration files required by the selected payload
- [x] 2.3 Determine how `loader.kboot` discovers the staged payload from a bootup/u-root initramfs environment

## 3. QEMU UEFI Proof

- [x] 3.1 Build a minimal QEMU UEFI command or script that runs bootup/u-root with `loader.kboot` available
- [x] 3.2 Run the loader manually from the stage-1 environment and capture the first failure or successful boot signal
- [x] 3.3 If the handoff succeeds, reduce the commands to a reproducible smoke procedure
  - Not applicable: the loader fails before `kexec_load` because it cannot obtain Linux `boot_params`/EFI map metadata.
- [x] 3.4 If the handoff fails, isolate whether the blocker is loader execution, firmware state, file visibility, or FreeBSD kernel handoff

## 4. Recommendation

- [x] 4.1 Record the spike outcome in docs or the OpenSpec design with exact commands and observed evidence
- [x] 4.2 Recommend either a future `freebsd-kboot` boot action change or continued deferral with blockers
- [x] 4.3 Keep FreeBSD and mfsBSD out of the default catalog unless the smoke evidence supports a follow-up implementation
