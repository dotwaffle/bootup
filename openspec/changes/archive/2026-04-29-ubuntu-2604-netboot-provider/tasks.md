## 1. Provider Shape

- [x] 1.1 Add an Ubuntu provider package with target metadata
- [x] 1.2 Add a boot plan for Ubuntu 26.04 amd64 netboot release URLs
- [x] 1.3 Register Ubuntu in the default provider set

## 2. Verification and Staging

- [x] 2.1 Allow HTTPS-only Ubuntu netboot staging by default
- [x] 2.2 Verify Ubuntu `SHA256SUMS` with `SHA256SUMS.gpg` when keyring material is supplied
- [x] 2.3 Verify kernel/initrd artifacts with explicit SHA-256 hashes when supplied
- [x] 2.4 Stage verified Ubuntu artifacts to disk

## 3. Tests and Documentation

- [x] 3.1 Cover Ubuntu target metadata and plan URLs
- [x] 3.2 Cover HTTPS-only and non-HTTPS fail-closed behavior
- [x] 3.3 Cover signed checksum and explicit artifact hash staging
- [x] 3.4 Document Ubuntu verification constraints and configuration

## 4. Validation

- [x] 4.1 Run Go tests, race tests, and vmtest compile path
- [x] 4.2 Run golangci-lint for default and fixture-tag builds
- [x] 4.3 Mark OpenSpec tasks complete after validation
