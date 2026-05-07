## 1. Specification

- [x] 1.1 Add OpenSpec deltas for remote signed policy sources, cache fallback,
      signing ergonomics, smoke coverage, and menu fallback.
- [x] 1.2 Validate the change with strict OpenSpec validation.

## 2. Signing Flow

- [x] 2.1 Add a tested policy signing helper or documented example flow.
- [x] 2.2 Update README and policy documentation with local and remote signed
      policy examples.

## 3. Remote Policy Source

- [x] 3.1 Add HTTPS policy URL loading with timeout handling.
- [x] 3.2 Add authenticated cache write and fallback behavior.
- [x] 3.3 Add CLI integration, validation, and redacted diagnostics.

## 4. Smoke And Fallback

- [x] 4.1 Add an end-to-end signed policy smoke that selects a real catalog
      target, validates options, and checks diagnostics posture.
- [x] 4.2 Add an explicit menu fallback option for policy failure.
- [x] 4.3 Cover policy fallback behavior with focused tests.

## 5. Verification

- [x] 5.1 Run gofmt.
- [x] 5.2 Run go test ./...
- [x] 5.3 Run golangci-lint run.
- [ ] 5.4 Archive the change and update the handoff file.
