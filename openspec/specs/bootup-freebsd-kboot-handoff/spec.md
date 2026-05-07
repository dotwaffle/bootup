# bootup-freebsd-kboot-handoff Specification

## Purpose
Define the evidence required before bootup treats FreeBSD `loader.kboot` as a
supported handoff path, and keep FreeBSD-shaped payloads out of the executable
catalog until that evidence exists.

## Requirements
### Requirement: FreeBSD kboot viability is evidence-based
Bootup SHALL treat FreeBSD `loader.kboot` support as an experimental
investigation until a reproducible QEMU UEFI handoff proves the loader can be
obtained, executed, and supplied with the required FreeBSD or mfsBSD artifacts.

#### Scenario: Candidate route is assessed
- **WHEN** the FreeBSD kboot handoff spike evaluates a candidate route
- **THEN** it SHALL record the `loader.kboot` source or build method, firmware
  assumptions, staged artifact layout, and exact FreeBSD or mfsBSD payloads
  used by the test

#### Scenario: Viability claim includes boot evidence
- **WHEN** the spike concludes that the handoff path is viable
- **THEN** it SHALL include a reproducible QEMU UEFI command or script and the
  observed boot signal that demonstrates control reached the FreeBSD or mfsBSD
  environment

#### Scenario: Blocker keeps targets deferred
- **WHEN** `loader.kboot` cannot be built, cannot run from the Linux/u-root
  stage, cannot see the staged artifacts, or cannot hand off to the target
  kernel
- **THEN** bootup SHALL keep FreeBSD and mfsBSD targets out of the executable
  default catalog and document the blocking condition

### Requirement: FreeBSD kboot artifacts are not vendored
Bootup SHALL NOT commit generated `loader.kboot` binaries, downloaded FreeBSD
or mfsBSD release payloads, generated initramfs images, or VM disk images as
part of the investigation.

#### Scenario: Experimental materials are generated locally
- **WHEN** the spike requires FreeBSD loader, kernel, module, root filesystem,
  or VM artifacts
- **THEN** those artifacts SHALL be downloaded or built into ignored local
  paths or `/tmp`, with enough commands documented to reproduce them
