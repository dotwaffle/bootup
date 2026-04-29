# Repository Guidelines

## Project Structure & Module Organization

Bootup is a Go module for a chainloaded Linux/u-root stage-1 netboot
environment.

- `cmd/bootup/`: main bootup binary entrypoint.
- `internal/`: application packages, including providers, runtime prep,
  handoff, UI, logging, and verification helpers.
- `internal/providers/debian/`: Debian trixie amd64 netboot provider.
- `scripts/`: helper scripts for initramfs and QEMU launch flows.
- `examples/`: sample iPXE and GRUB chainload configuration.
- `docs/`: architecture and operational notes.
- `test/vmtest/`: tagged VM integration tests.
- `openspec/`: active OpenSpec change artifacts and requirements.

## Build, Test, and Development Commands

Run normal checks before submitting changes:

```sh
golangci-lint run
go test ./...
go test -tags vmtest ./test/vmtest
go build -trimpath -o /tmp/bootup ./cmd/bootup
```

Use `scripts/build-initramfs.sh` to build the u-root initramfs and
`scripts/run-qemu.sh` to boot it with QEMU. The VM tests are tagged because
they require QEMU/kernel setup.

## Coding Style & Naming Conventions

Use idiomatic Go formatted with `gofmt`. Keep package names short and avoid
stutter, for example `provider.Registry`, not `provider.ProviderRegistry`.
Prefer explicit dependencies through constructors/config structs over hidden
globals. Use structured logging with `log/slog`.

Linting is enforced by `.golangci.yaml`; do not weaken lint rules to avoid
fixing real issues.

## Testing Guidelines

Use Go’s standard `testing` package. Prefer table-driven tests for multiple
cases and call `t.Parallel()` where tests are independent. Test names should
describe behavior, for example `TestVerifyArtifactChecksumRejectsMismatch`.

Keep tests hermetic by default. Network, QEMU, or host-kernel tests should be
behind tags or explicit environment requirements.

## Commit & Pull Request Guidelines

There is no committed project history yet. Use the repository convention:
Linux-kernel-style subjects, `subsystem: imperative summary`. Do not use
Conventional Commits and do not use `chore`.

Examples:

```text
provider: add Debian netboot planning
ci: run golangci-lint in pull requests
```

Pull requests should include a short problem statement, implementation summary,
and verification results.

## Security & Configuration Notes

Do not commit distro keyrings, generated binaries, initramfs images, or other
binary payloads. TLS roots are embedded through `github.com/breml/rootcerts`;
distribution signature verification should use explicit caller-supplied trust
material.
