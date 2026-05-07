## 1. Kernel Config Prerequisites

- [x] 1.1 Add `CONFIG_KALLSYMS_ALL=y` and `CONFIG_PROC_KCORE=y` to the bootup amd64 kernel fragment
- [x] 1.2 Extend `scripts/check-kernel-config.sh` to require the FreeBSD kboot metadata symbols
- [x] 1.3 Update kernel config fixtures and tests so missing or modular kboot prerequisites fail clearly

## 2. FreeBSD Kboot Smoke

- [x] 2.1 Add an opt-in smoke script or documented helper that stages `loader.kboot` and the FreeBSD bootonly ISO outside tracked paths
- [x] 2.2 Build the smoke around virtio-block payload presentation and the manual `bootdev=/dev/vda:` loader command
- [x] 2.3 Capture success and failure markers that distinguish reaching the FreeBSD loader menu, clearing the metadata blocker, and reaching the target environment

## 3. Documentation And Validation

- [x] 3.1 Document the kernel prerequisite rationale and the FreeBSD kboot smoke invocation
- [x] 3.2 Run OpenSpec validation and focused kernel config tests
- [x] 3.3 Confirm no generated kernels, FreeBSD artifacts, initramfs images, ISOs, or VM disks are tracked
