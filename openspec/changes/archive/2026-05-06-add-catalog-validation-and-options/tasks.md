## 1. Catalog Option Model

- [x] 1.1 Add tests for catalog option source validation
- [x] 1.2 Add target option metadata types to provider/catalog models
- [x] 1.3 Extend catalog source generation and embedded catalog JSON with option metadata
- [x] 1.4 Reject duplicate option IDs, unsupported option types, invalid values, and malformed command-line fragments

## 2. Option Planning

- [x] 2.1 Add tests for selected option validation and command-line expansion
- [x] 2.2 Add CLI/app input for selecting target options by option ID and value
- [x] 2.3 Pass selected options through explicit provider planning input
- [x] 2.4 Append option command-line fragments after provider defaults and before `--append-cmdline`

## 3. Catalog Discovery

- [x] 3.1 Add tests for compact catalog list output containing target ID, distro, release, architecture, provider, and action
- [x] 3.2 Add or extend target show output with metadata, artifacts, lifecycle decoration, and option definitions
- [x] 3.3 Return a clear error for unknown target show requests without staging artifacts
- [x] 3.4 Update docs for list/show usage and option metadata

## 4. Live Smoke Validation

- [x] 4.1 Add tests or scripts for selecting live smoke targets by catalog target ID
- [x] 4.2 Remove MemTest86+ live smoke coverage until a compatible handoff exists
- [x] 4.3 Add kernel+initrd live smoke coverage for one generic Linux catalog target
- [x] 4.4 Gate live smoke coverage behind explicit tags or environment variables
- [x] 4.5 Document QEMU, network, and host-kernel requirements for live smoke runs

## 5. Verification

- [x] 5.1 Run `go generate ./internal/catalog`
- [x] 5.2 Run gofmt on touched Go files
- [x] 5.3 Run `go test ./...`
- [x] 5.4 Run `go vet ./...`
- [x] 5.5 Run `golangci-lint run`
- [x] 5.6 Run `go build -trimpath -o /tmp/bootup ./cmd/bootup`
