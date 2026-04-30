## 1. Release Assembly

- [x] 1.1 Add a release assembly script that builds or collects the bootup binary, kernel image/config, zstd initramfs, and hybrid ISO into `dist/release/`.
- [x] 1.2 Implement stable release artifact names that include the bootup release version, target architecture, and Linux kernel version for kernel outputs.
- [x] 1.3 Generate a versioned JSON manifest with artifact roles, names, byte sizes, SHA-256 digests, git commit, architecture, and default trust-material posture.
- [x] 1.4 Generate a `SHA256SUMS` file covering every public binary artifact and the manifest.
- [x] 1.5 Ensure the default release initramfs and ISO are built without distribution-specific archive keyrings or trust bundles.

## 2. Release Validation

- [x] 2.1 Add a release artifact validation script that checks required artifact presence, manifest fields, checksum integrity, and public artifact names.
- [x] 2.2 Extend ISO validation to assert the bootup kernel, bootup initramfs, GRUB config, and UEFI fallback boot path are present.
- [x] 2.3 Add or wire a BIOS ISO smoke command suitable for release validation.
- [x] 2.4 Include new release scripts in shell syntax checks.

## 3. Release Automation

- [x] 3.1 Add a release workflow that runs on release tags and manual dispatch.
- [x] 3.2 Install the release build dependencies in CI and run the release assembly and validation scripts.
- [x] 3.3 Publish validated release artifacts for tag builds.
- [x] 3.4 Keep release publication permissions scoped to the release workflow, not normal pull-request CI.

## 4. Operator Documentation

- [x] 4.1 Add release documentation that maps artifacts to iPXE, GRUB, and ISO boot paths.
- [x] 4.2 Document checksum and manifest verification before booting downloaded artifacts.
- [x] 4.3 Document release build dependencies, local release rehearsal, and the default no-distribution-keyring posture.
- [x] 4.4 Link release documentation from the existing launch and boot media docs.

## 5. Verification

- [x] 5.1 Run `openspec validate --all`.
- [x] 5.2 Run shell syntax checks for all release/build/QEMU scripts.
- [x] 5.3 Run `go test -race ./...` and `go test -race -tags bootup_debian_fixture ./...`.
- [x] 5.4 Run `golangci-lint run` and `golangci-lint run --build-tags bootup_debian_fixture`.
- [x] 5.5 Build and validate a local release bundle, including the configured ISO smoke gate.
