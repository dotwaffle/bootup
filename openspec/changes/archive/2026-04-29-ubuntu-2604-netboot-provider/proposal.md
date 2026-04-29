## Why

Debian now proves the end-to-end bootup flow. Ubuntu 26.04 is the next useful
catalog entry, but its release checksum metadata covers ISO artifacts rather
than the extracted netboot kernel and initrd, so the provider must avoid unsafe
implicit trust.

## What Changes

- Add a build-time Ubuntu provider exposing Ubuntu 26.04 amd64 netboot.
- Generate a boot plan using the official 26.04 release netboot kernel/initrd
  URLs and live-server ISO URL.
- Stage Ubuntu netboot artifacts over HTTPS by default, with optional
  caller-supplied OpenPGP trust material and explicit netboot artifact hashes
  for stronger verification.
- Register Ubuntu alongside Debian in the default provider set.
- Add unit tests for target metadata, planning, verification, and fail-closed
  staging behavior.

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `bootup-netboot`: Adds Ubuntu 26.04 amd64 netboot as a build-time provider
  target with explicit verification requirements.

## Impact

- Adds `internal/providers/ubuntu`.
- Updates default provider registration and catalog output.
- Does not commit Ubuntu keyrings or binary artifacts.
- Uses official Ubuntu release URLs for 26.04 amd64 netboot planning.
