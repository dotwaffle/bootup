## 1. Real Debian Boot Smoke

- [x] 1.1 Add a local command path that generates ignored Debian trust source and builds a Debian-capable initramfs
- [x] 1.2 Add a QEMU smoke path for selecting Debian and attempting kexec into the Debian Installer
- [x] 1.3 Document expected local failure modes: missing keyring, no network, no QEMU, and host kexec permission failures

## 2. Repeatable Validation

- [x] 2.1 Add an opt-in live Debian smoke test that skips unless all required environment inputs are present
- [x] 2.2 Keep hermetic fixture coverage in default/CI-friendly checks
- [x] 2.3 Document exact commands for local real-smoke and fixture-smoke runs

## 3. Serial Operator Menu

- [x] 3.1 Render stable numeric selection indexes in the target list
- [x] 3.2 Add visible status rendering for planning, staging, verifying, and loading phases
- [x] 3.3 Add a readable failure screen for plan/stage/kexec failures
- [x] 3.4 Cover menu selection and failure rendering with unit tests

## 4. Provider Catalog Shape

- [x] 4.1 Add provider target catalog metadata for distribution, release, architecture, and kind
- [x] 4.2 Populate Debian target catalog metadata
- [x] 4.3 Render catalog metadata in the text UI without breaking 80-column output
- [x] 4.4 Cover catalog metadata with provider and UI tests
