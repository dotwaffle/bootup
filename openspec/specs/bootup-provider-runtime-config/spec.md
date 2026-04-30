# bootup-provider-runtime-config Specification

## Purpose
Define bootup's operator-supplied runtime configuration for compiled-in
providers, including source URL overrides and verification material.

## Requirements
### Requirement: Operator provider runtime configuration
Bootup SHALL allow operators to supply provider source and verification inputs
for compiled-in providers through an explicit runtime configuration file.

#### Scenario: Provider config is absent
- **WHEN** bootup starts without a provider runtime configuration file
- **THEN** bootup SHALL preserve the compiled-in provider defaults

#### Scenario: Provider config is loaded
- **WHEN** bootup starts with a readable provider runtime configuration file
- **THEN** bootup SHALL apply configured provider source URLs, keyring paths,
  and artifact hash pins before target discovery

#### Scenario: Provider keyring path is configured
- **WHEN** a provider runtime configuration entry references a keyring path
- **THEN** bootup SHALL read that keyring from the local filesystem and pass its
  bytes to the compiled-in provider

#### Scenario: Provider config is invalid
- **WHEN** the provider runtime configuration file is malformed, references an
  unknown provider, includes an invalid hash pin, or references unreadable trust
  material
- **THEN** bootup SHALL fail startup before provider target discovery or artifact
  retrieval

#### Scenario: Release artifacts remain provider-neutral
- **WHEN** bootup is built with the default release packaging flow
- **THEN** the release artifacts MUST NOT embed distribution-specific archive
  keyrings or trust bundles
