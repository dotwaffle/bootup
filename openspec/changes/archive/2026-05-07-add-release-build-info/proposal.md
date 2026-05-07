## Why

Release artifacts identify their source commit in the manifest, but the
standalone bootup binary cannot identify the release metadata stamped into it.
That leaves operators with an avoidable gap when debugging a booted initramfs,
standalone binary, or copied artifact outside the release directory.

## What Changes

- Add a `bootup --version` CLI path that prints the bootup release version,
  git commit, build date, source tree state, and Go runtime version.
- Stamp release builds with the release version, commit, build date, and dirty
  state through Go linker variables.
- Extend the release manifest and validation script so the published manifest
  records and checks the stamped bootup binary metadata.
- Document how operators can inspect binary build metadata from release
  artifacts.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `bootup-release-packaging`: release artifacts expose and validate stamped
  bootup binary build metadata.

## Impact

- Affected code: `cmd/bootup`, a small internal build-info package,
  `scripts/build-release.sh`, `scripts/check-release-artifacts.sh`, and release
  documentation.
- Public behavior: `bootup --version` becomes a stable diagnostic command.
- Dependencies: no new third-party dependencies.
