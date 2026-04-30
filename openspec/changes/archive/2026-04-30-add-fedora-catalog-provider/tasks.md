## 1. OpenSpec And Docs

- [x] 1.1 Add OpenSpec deltas for Fedora provider, generated catalogs, lifecycle source, shared helpers, and hosted catalog design.
- [x] 1.2 Refresh README/docs for Fedora targets, catalog generation, lifecycle source, hosted catalog requirements, and smoke coverage.

## 2. Catalog Generation And Lifecycle Source

- [x] 2.1 Add a structured default catalog source file.
- [x] 2.2 Add `go generate ./internal/catalog` tooling that writes deterministic `default.json`.
- [x] 2.3 Preserve target `source` and `lifecycle` metadata through generation.
- [x] 2.4 Add tests that fail when generated output is stale.

## 3. Fedora Provider

- [x] 3.1 Add Fedora runtime config fields and validation for release URL and optional kernel/initrd SHA-256 pins.
- [x] 3.2 Add a Fedora Server amd64 netboot provider with static target planning.
- [x] 3.3 Add Fedora artifact staging with HTTPS-only default and optional SHA-256 verification.
- [x] 3.4 Register Fedora in the default provider set and embedded catalog.

## 4. Shared Helper Cleanup

- [x] 4.1 Add tested shared provider HTTP helpers for fetch/status/probe/path behavior.
- [x] 4.2 Refactor Debian and Ubuntu discovery to use the shared helpers.

## 5. Tests And Smoke Coverage

- [x] 5.1 Add unit coverage for Fedora targets, planning, staging, runtime config, and catalog registration.
- [x] 5.2 Add catalog and VM expectations for Fedora default targets.
- [x] 5.3 Add fixture/smoke coverage hooks that compile Fedora provider paths without live network requirements.

## 6. Validation

- [x] 6.1 Run gofmt/go generate, full Go tests, race tests, tagged tests, lint, build, OpenSpec validation, and diff checks.
- [x] 6.2 Archive the OpenSpec change, commit, push, check CI, and update the persistent handoff.
