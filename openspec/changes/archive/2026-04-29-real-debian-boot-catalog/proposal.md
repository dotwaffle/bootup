## Why

Bootup can already stage verified Debian artifacts, but the current repository
stops short of a repeatable real Debian boot smoke, a comfortable serial
selection workflow, and provider metadata that can grow toward Salstar or
netboot.xyz-style catalogs. The next slice should prove the real Debian path
and make that path easy to repeat without committing trust material.

## What Changes

- Add documented local commands for generating ignored Debian trust material,
  building a Debian-capable initramfs, and booting it under QEMU.
- Add an opt-in real Debian smoke test that validates live metadata/artifact
  staging when QEMU, network, and local keyring inputs are available.
- Improve the serial menu with indexed selection, status rendering, and a
  readable failure screen.
- Extend provider targets with catalog metadata for grouping by distribution,
  release, architecture, and target kind.

## Capabilities

### Modified Capabilities

- `bootup-netboot`: Adds real Debian smoke coverage, richer serial selection,
  and catalog metadata for provider targets.

## Impact

- Keeps the default repository binary free of committed or embedded Debian
  keyrings while allowing local single-binary Debian-capable builds.
- Adds optional network/QEMU checks that are skipped unless explicitly enabled.
- Prepares the provider model for a larger catalog without adding additional
  distributions in this change.
