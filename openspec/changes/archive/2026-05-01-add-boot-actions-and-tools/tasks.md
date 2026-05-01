## 1. Boot Action Model

- [x] 1.1 Add boot action and static Linux source metadata tests
- [x] 1.2 Implement provider target and boot plan action metadata
- [x] 1.3 Allow optional initrd artifacts in app planning, staging, and handoff

## 2. Local Boot

- [x] 2.1 Add local boot provider and handoff tests
- [x] 2.2 Implement local boot provider and action-dispatched executor
- [x] 2.3 Include the u-root boot applet in generated initramfs images

## 3. Generic Linux Provider

- [x] 3.1 Add generic Linux provider planning and staging tests
- [x] 3.2 Implement the generic Linux provider
- [x] 3.3 Register the generic Linux provider in the default provider set

## 4. Catalog Expansion

- [x] 4.1 Add catalog generation tests for local boot and Linux utility targets
- [x] 4.2 Add default source entries for openSUSE, Arch Linux, GParted Live, and MemTest86+
- [x] 4.3 Regenerate the embedded default catalog

## 5. Operator Options

- [x] 5.1 Add tests for command-line append behavior
- [x] 5.2 Add CLI and app support for appending command-line parameters
- [x] 5.3 Add tests for explicit network configuration
- [x] 5.4 Add CLI and runtime support for interface address, route, and DNS setup

## 6. Documentation

- [x] 6.1 Document local boot, generic Linux targets, command-line append, and network flags
- [x] 6.2 Document BSD, HDT, memdisk ISO, and chainload support as deferred

## 7. Verification

- [x] 7.1 Run go generate for the catalog
- [x] 7.2 Run gofmt, go test, go vet, golangci-lint, and go build
