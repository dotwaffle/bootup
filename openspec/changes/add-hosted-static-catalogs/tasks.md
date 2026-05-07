## 1. Catalog Metadata And Trust

- [x] 1.1 Add catalog document freshness metadata parsing and validation tests.
- [x] 1.2 Implement optional top-level `published_at` and `expires_at` fields on static catalog documents.
- [x] 1.3 Add hosted catalog authentication tests for SHA-256 pins and Ed25519 detached signatures.
- [x] 1.4 Implement hosted catalog trust verification with fail-closed errors.

## 2. Hosted Loading And Cache

- [x] 2.1 Add hosted catalog fetch tests for HTTPS success, unsupported schemes, HTTP errors, and response-size limits.
- [x] 2.2 Implement context-bound hosted catalog fetching and authenticated parsing.
- [x] 2.3 Add cache tests for successful cache writes, opt-in fallback, stale cache rejection, and unauthenticated cache rejection.
- [x] 2.4 Implement cache write and fallback behavior after hosted catalog validation.

## 3. CLI And Operator Documentation

- [x] 3.1 Add command startup tests for hosted catalog flags, local/hosted mutual exclusion, missing trust config, and cache fallback.
- [x] 3.2 Wire hosted catalog source flags into `cmd/bootup` while preserving existing embedded and local catalog behavior.
- [x] 3.3 Document hosted static catalog publication, trust, freshness, replacement semantics, cache fallback, and artifact-trust separation.

## 4. Validation

- [x] 4.1 Run `go test ./...` and `go test -tags bootup_debian_fixture ./...`.
- [x] 4.2 Run `golangci-lint run` and `golangci-lint run --build-tags bootup_debian_fixture`.
- [x] 4.3 Run `go build -trimpath -o /tmp/bootup ./cmd/bootup` and fixture-tag build.
- [x] 4.4 Run `openspec validate add-hosted-static-catalogs` and `openspec validate --all`.
- [x] 4.5 Confirm no binary artifacts are tracked before commit.
