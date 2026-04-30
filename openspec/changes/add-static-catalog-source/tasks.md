## 1. Catalog Source

- [x] 1.1 Add tests for parsing, validating, and filtering static catalog documents.
- [x] 1.2 Implement the static catalog loader with embedded default JSON and local file loading.
- [x] 1.3 Add the default catalog with Debian bookworm, Debian trixie, and Ubuntu 26.04 amd64 netboot targets.

## 2. Provider Wiring

- [x] 2.1 Add tests for command startup using embedded and local catalog sources.
- [x] 2.2 Pass validated catalog targets into compiled-in provider registration.
- [x] 2.3 Add tests for Debian release-specific planning from configured catalog targets.
- [x] 2.4 Generalize Debian provider planning for configured amd64 release targets.

## 3. Documentation

- [x] 3.1 Document the static catalog JSON format and `--catalog` replacement behavior.
- [x] 3.2 Document hosted catalog and dynamic discovery boundaries.

## 4. Validation

- [x] 4.1 Run Go tests, fixture tests, vmtest coverage, build, lint, and OpenSpec validation.
- [ ] 4.2 Archive the OpenSpec change, commit logical patches, push, and update the persistent handoff.
