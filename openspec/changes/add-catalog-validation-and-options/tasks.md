## 1. Catalog Option Model

- [ ] 1.1 Add tests for catalog option source validation
- [ ] 1.2 Add target option metadata types to provider/catalog models
- [ ] 1.3 Extend catalog source generation and embedded catalog JSON with option metadata
- [ ] 1.4 Reject duplicate option IDs, unsupported option types, invalid values, and malformed command-line fragments

## 2. Option Planning

- [ ] 2.1 Add tests for selected option validation and command-line expansion
- [ ] 2.2 Add CLI/app input for selecting target options by option ID and value
- [ ] 2.3 Pass selected options through explicit provider planning input
- [ ] 2.4 Append option command-line fragments after provider defaults and before `--append-cmdline`

## 3. Catalog Discovery

- [ ] 3.1 Add tests for compact catalog list output containing target ID, distro, release, architecture, provider, and action
- [ ] 3.2 Add or extend target show output with metadata, artifacts, lifecycle decoration, and option definitions
- [ ] 3.3 Return a clear error for unknown target show requests without staging artifacts
- [ ] 3.4 Update docs for list/show usage and option metadata

## 4. Live Smoke Validation

- [ ] 4.1 Add tests or scripts for selecting live smoke targets by catalog target ID
- [ ] 4.2 Add kernel-only live smoke coverage for the MemTest86+ catalog target
- [ ] 4.3 Add kernel+initrd live smoke coverage for one generic Linux catalog target
- [ ] 4.4 Gate live smoke coverage behind explicit tags or environment variables
- [ ] 4.5 Document QEMU, network, and host-kernel requirements for live smoke runs

## 5. Verification

- [ ] 5.1 Run `go generate ./internal/catalog`
- [ ] 5.2 Run gofmt on touched Go files
- [ ] 5.3 Run `go test ./...`
- [ ] 5.4 Run `go vet ./...`
- [ ] 5.5 Run `golangci-lint run`
- [ ] 5.6 Run `go build -trimpath -o /tmp/bootup ./cmd/bootup`
