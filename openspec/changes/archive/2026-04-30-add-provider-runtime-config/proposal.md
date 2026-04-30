## Why

Default bootup artifacts intentionally ship without distribution-specific
archive keyrings or trust bundles, but the default binary has no runtime path
for operators to supply provider verification material. That leaves verified
provider paths dependent on custom application builds and does not scale to many
future bootable distro image sources.

## What Changes

- Add an operator-supplied provider configuration file loaded by the bootup
  command at startup.
- Allow the configuration to provide provider-specific source URLs, OpenPGP
  keyring paths, and explicit artifact hash pins for compiled-in providers.
- Keep defaults provider-neutral: no distro keyrings are committed, packaged, or
  embedded in release artifacts.
- Fail fast when a configured provider entry, trust-material path, or hash value
  is invalid.
- Preserve default provider discovery behavior when no runtime provider
  configuration is supplied.

## Capabilities

### New Capabilities

- `bootup-provider-runtime-config`: operator-supplied runtime configuration for
  compiled-in provider sources and verification material.

### Modified Capabilities

- `bootup-netboot`: provider verification material can be supplied through the
  operator runtime configuration as well as application-level provider config.

## Impact

- Affects `cmd/bootup` startup flags and provider registration.
- Adds a small internal configuration loader with no new runtime dependency.
- Updates Debian and Ubuntu default provider wiring to consume operator-supplied
  keyring paths, source URL overrides, and Ubuntu netboot SHA-256 pins.
- Updates launch/security documentation and OpenSpec requirements for the new
  runtime configuration path.
