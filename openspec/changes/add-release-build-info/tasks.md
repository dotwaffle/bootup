## 1. Build Info CLI

- [x] 1.1 Add build-info tests for default and stamped metadata rendering.
- [x] 1.2 Add a `bootup --version` CLI test that proves version reporting bypasses normal catalog/provider startup.
- [x] 1.3 Implement build-info metadata and wire `bootup --version`.

## 2. Release Stamping

- [x] 2.1 Add release script/manifest tests for linker stamping and manifest metadata fields.
- [x] 2.2 Stamp release builds with version, commit, build date, and source tree state.
- [x] 2.3 Record stamped bootup binary metadata in release manifests.
- [x] 2.4 Validate release manifests against `bootup --version` output.

## 3. Documentation and Verification

- [x] 3.1 Document release binary build metadata inspection.
- [x] 3.2 Run required Go, lint, OpenSpec, and diff checks.
