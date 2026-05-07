## 1. Specification

- [x] 1.1 Add OpenSpec deltas for remote signed policy sources, cache fallback,
      signing ergonomics, smoke coverage, and menu fallback.
- [x] 1.2 Validate the change with strict OpenSpec validation.

## 2. Signing Flow

- [ ] 2.1 Add a tested policy signing helper or documented example flow.
- [ ] 2.2 Update README and policy documentation with local and remote signed
      policy examples.

## 3. Remote Policy Source

- [ ] 3.1 Add HTTPS policy URL loading with timeout handling.
- [ ] 3.2 Add authenticated cache write and fallback behavior.
- [ ] 3.3 Add CLI integration, validation, and redacted diagnostics.

## 4. Smoke And Fallback

- [ ] 4.1 Add an end-to-end signed policy smoke that selects a real catalog
      target, validates options, and checks diagnostics posture.
- [ ] 4.2 Add an explicit menu fallback option for policy failure.
- [ ] 4.3 Cover policy fallback behavior with focused tests.

## 5. Verification

- [ ] 5.1 Run gofmt.
- [ ] 5.2 Run go test ./...
- [ ] 5.3 Run golangci-lint run.
- [ ] 5.4 Archive the change and update the handoff file.
