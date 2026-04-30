## Why

Bootup now has a larger static catalog, but static entries still require a tool
or catalog update before new distro releases, architectures, or install options
appear. Mode-2 dynamic discovery needs a provider contract so compiled-in
providers can discover selectable concrete targets at runtime without becoming
fully dynamic policy execution.

## What Changes

- Define a dynamic distro discovery capability for compiled-in providers.
- Add a discovery flow that lists distro families first, then discovers
  releases, architectures, variants, and install options when a family is
  selected.
- Define discovered target metadata as data returned by compiled-in provider
  logic, not provider code loaded from catalogs or the network.
- Allow optional lifecycle decoration such as supported/obsolete/EOL status
  when the provider can obtain it from configured data sources.
- Keep hosted catalog loading and mode-3 policy/script execution out of scope.

## Capabilities

### New Capabilities
- `bootup-dynamic-distro-discovery`: Runtime discovery of concrete targets by
  compiled-in provider logic after an operator selects a distro family.

### Modified Capabilities
- `bootup-static-catalog-source`: Clarify that static catalogs remain concrete
  target lists and do not perform dynamic discovery.
- `bootup-netboot`: Add provider discovery behavior that complements static
  target listing without requiring runtime provider plugins.

## Impact

- Adds provider discovery interfaces, lifecycle decoration metadata, menu and
  diagnostic discovery flows, and Debian amd64 netboot discovery.
- Affects provider interfaces, operator UI flows, provider runtime
  configuration, docs, and tests.
- No hosted catalog URL loading, script execution, self-hosted policy server, or
  release tagging is introduced by this proposal.
