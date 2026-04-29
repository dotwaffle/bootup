## 1. Ubuntu Live Staging Test

- [x] 1.1 Add an opt-in live Ubuntu staging test
- [x] 1.2 Ensure the test skips unless explicitly enabled
- [x] 1.3 Assert staged Ubuntu kernel and initrd files are non-empty

## 2. QEMU Smoke Helper

- [x] 2.1 Add `scripts/smoke-real-ubuntu.sh`
- [x] 2.2 Build a normal initramfs that selects `ubuntu-2604-amd64-netboot`
- [x] 2.3 Configure QEMU user networking for host-kernel fallback
- [x] 2.4 Attempt kexec into Ubuntu netboot

## 3. Documentation

- [x] 3.1 Document the Ubuntu live staging test command
- [x] 3.2 Document the Ubuntu QEMU smoke command
- [x] 3.3 Document the HTTPS-only trust model for this smoke

## 4. Validation

- [x] 4.1 Run shell syntax checks
- [x] 4.2 Run Go tests, race tests, vmtest compile path, and linters
- [x] 4.3 Mark tasks complete after validation
