# bootup-provider-runtime-config Specification

## Purpose
Define bootup's operator-supplied runtime configuration for compiled-in
providers, including source URL overrides and verification material.
## Requirements
### Requirement: Operator provider runtime configuration
Bootup SHALL allow operators to supply provider source, discovery, lifecycle,
and verification inputs for compiled-in providers through an explicit runtime
configuration file.

#### Scenario: Provider config is absent
- **WHEN** bootup starts without a provider runtime configuration file
- **THEN** bootup SHALL preserve the compiled-in provider defaults

#### Scenario: Provider config is loaded
- **WHEN** bootup starts with a readable provider runtime configuration file
- **THEN** bootup SHALL apply configured provider source URLs, keyring paths,
  artifact hash pins, discovery settings, and lifecycle metadata before target
  discovery

#### Scenario: Provider discovery config is supplied
- **WHEN** provider runtime configuration includes discovery URL and discovery
  timeout fields for a compiled-in provider
- **THEN** bootup SHALL validate those fields and pass them to that provider
  before discovery can run

#### Scenario: Provider lifecycle config is supplied
- **WHEN** provider runtime configuration includes lifecycle metadata for a
  provider release
- **THEN** bootup SHALL validate lifecycle status, source, and date fields
  before provider registration

#### Scenario: Provider keyring path is configured
- **WHEN** a provider runtime configuration entry references a keyring path
- **THEN** bootup SHALL read that keyring from the local filesystem and pass its
  bytes to the compiled-in provider

#### Scenario: Fedora provider config is supplied
- **WHEN** provider runtime configuration includes Fedora release URL or
  kernel/initrd hash pins
- **THEN** bootup SHALL validate those fields and pass them to the Fedora
  provider before target planning or artifact staging

#### Scenario: Provider config is invalid
- **WHEN** the provider runtime configuration file is malformed, references an
  unknown provider, includes an invalid hash pin, or references unreadable trust
  material
- **THEN** bootup SHALL fail startup before provider target discovery or artifact
  retrieval

#### Scenario: Provider discovery config is invalid
- **WHEN** discovery URL, discovery timeout, lifecycle status, or lifecycle date
  configuration is malformed
- **THEN** bootup SHALL fail startup before registering provider targets

#### Scenario: Fedora provider config is invalid
- **WHEN** Fedora release URL or hash pin configuration is malformed
- **THEN** bootup SHALL fail startup before registering provider targets

#### Scenario: Release artifacts remain provider-neutral
- **WHEN** bootup is built with the default release packaging flow
- **THEN** the release artifacts MUST NOT embed distribution-specific archive
  keyrings or trust bundles

### Requirement: Startup command-line append configuration
Bootup SHALL expose a startup option for appending operator-supplied kernel
parameters to selected boot plans.

#### Scenario: Append option is absent
- **WHEN** bootup starts without command-line append configuration
- **THEN** provider command lines SHALL remain unchanged

#### Scenario: Append option is present
- **WHEN** bootup starts with command-line append configuration
- **THEN** bootup SHALL append the configured text to the selected boot plan
  command line

### Requirement: Startup network configuration
Bootup SHALL expose startup options for interface, address, default gateway,
and DNS configuration.

#### Scenario: Address and gateway are supplied
- **WHEN** bootup starts with interface address and default gateway
  configuration
- **THEN** it SHALL configure the link and default route before provider
  operations

#### Scenario: DNS servers are supplied
- **WHEN** bootup starts with DNS server configuration
- **THEN** it SHALL write resolver configuration before provider operations

