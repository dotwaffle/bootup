## 1. PTY Coverage

- [x] 1.1 Add a go-expect PTY test for `--mode=menu --ui=rich`
- [x] 1.2 Assert keyboard input selects the intended target
- [x] 1.3 Keep plain fallback tests deterministic

## 2. UI Polish

- [x] 2.1 Strengthen provider grouping and selected-row treatment
- [x] 2.2 Add a compact animated bootup banner
- [x] 2.3 Keep output serial-safe and 80x25 conscious

## 3. Size And QEMU Smoke

- [x] 3.1 Measure bootup binary size
- [x] 3.2 Measure menu-mode initramfs raw and zstd sizes
- [x] 3.3 Run QEMU menu smoke for `--ui=auto`
- [x] 3.4 Document the measured results and smoke command

## 4. Validation

- [x] 4.1 Run tests, race tests, vmtest compile path, and linters
- [x] 4.2 Mark tasks complete after validation
