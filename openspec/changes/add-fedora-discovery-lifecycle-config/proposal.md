## Why

Provider discovery lifecycle decoration is operator-controlled for Debian and
Ubuntu, but Fedora discovery cannot receive the same release lifecycle map
through provider runtime configuration. That leaves Fedora discovered targets
less useful in menus, diagnostics, and conformance output even though the
lifecycle model is already provider-neutral.

## What Changes

- Accept Fedora `lifecycle` entries in provider runtime configuration.
- Pass the validated Fedora lifecycle map into the Fedora provider during
  registration.
- Attach matching lifecycle entries to Fedora discovered targets, falling back
  to informational `unknown` metadata when no entry is configured.
- Update docs and tests for Fedora lifecycle discovery parity.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `bootup-provider-runtime-config`: Fedora provider config accepts lifecycle
  metadata using the existing provider lifecycle validation rules.
- `bootup-dynamic-distro-discovery`: Fedora discovered targets can carry
  configured lifecycle decoration.

## Impact

- Affected code: `internal/providerconfig`, `internal/providers/fedora`,
  provider registration, docs, and tests.
- Public behavior: operators can add Fedora release lifecycle entries in
  `--provider-config`.
- Dependencies: no new dependencies.
