## Why

Debian and Ubuntu can already discover concrete netboot targets at runtime, but
Fedora remains static-only even though its release tree follows a predictable
layout. Adding Fedora discovery broadens dynamic provider coverage while keeping
artifact staging behind explicit target selection.

## What Changes

- Add a Fedora discovery family that lists available amd64 Server netboot
  targets from a configured Fedora releases index.
- Add Fedora discovery URL and timeout runtime configuration.
- Keep discovery hermetic in default tests by using fake HTTP fixtures.
- Document the new discovery mode and provider configuration fields.

## Capabilities

### New Capabilities

### Modified Capabilities
- `bootup-dynamic-distro-discovery`: add Fedora dynamic discovery behavior.
- `bootup-provider-runtime-config`: add Fedora discovery URL and timeout
  configuration.
- `bootup-netboot`: include Fedora in the discoverable provider families.

## Impact

- Affected code: `internal/providers/fedora`, `internal/providerconfig`,
  default provider registration, CLI tests, docs, and OpenSpec.
- APIs: Fedora provider config gains `DiscoveryURL` and `DiscoveryTimeout`.
- Dependencies: no new external dependencies.
- Systems: discovery remains explicit through `discover-targets` or menu family
  selection; default target listing and staging behavior are unchanged.
