# bootup-freebsd-kboot-handoff Specification

## Purpose
Define the evidence required before bootup treats FreeBSD `loader.kboot` as a
supported handoff path, and constrain executable catalog targets to routes with
reproducible target-environment boot evidence.
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
- **THEN** bootup SHALL keep targets for that blocked route out of the
  executable default catalog and document the blocking condition

#### Scenario: Memory-root evidence enables mfsBSD catalog target
- **WHEN** the FreeBSD kboot smoke proves an mfsBSD memory-root payload reaches
  the target environment without target-visible root media
- **THEN** bootup MAY expose an mfsBSD `freebsd-kboot` target in the executable
  default catalog, provided all FreeBSD and mfsBSD payloads are downloaded and
  verified at runtime instead of vendored

### Requirement: mfsBSD kboot target stages verified runtime artifacts
Bootup SHALL stage executable mfsBSD `freebsd-kboot` targets from verified
runtime downloads and SHALL present the extracted mfsBSD root tree through
Linux hostfs.

#### Scenario: mfsBSD target stages loader and memory-root payload
- **WHEN** bootup stages an mfsBSD `freebsd-kboot` target
- **THEN** it SHALL verify a pinned mfsBSD ISO hash, extract the ISO contents
  from Linux, normalize compressed `kernel` and `mfsroot` payload files when
  needed, verify a pinned FreeBSD base archive hash, extract `loader.kboot` and
  `loader.help.kboot`, and prepare loader arguments containing `hostfs_root`,
  `bootdev=host:/`, and serial console settings

### Requirement: FreeBSD kboot artifacts are not vendored
Bootup SHALL NOT commit generated `loader.kboot` binaries, downloaded FreeBSD
or mfsBSD release payloads, generated initramfs images, or VM disk images as
part of the investigation.

#### Scenario: Experimental materials are generated locally
- **WHEN** the spike requires FreeBSD loader, kernel, module, root filesystem,
  or VM artifacts
- **THEN** those artifacts SHALL be downloaded or built into ignored local
  paths or `/tmp`, with enough commands documented to reproduce them

### Requirement: FreeBSD kboot smoke proves kernel metadata prerequisites
Bootup SHALL require an opt-in QEMU smoke before treating the FreeBSD kboot
handoff path as viable, and that smoke SHALL use a bootup kernel that exposes
the Linux metadata interfaces required by FreeBSD `loader.kboot`.

#### Scenario: Smoke includes kernel prerequisite evidence
- **WHEN** the FreeBSD kboot smoke is run
- **THEN** it SHALL verify or document that the bootup kernel exposes
  `CONFIG_DEBUG_KERNEL`, `CONFIG_KALLSYMS`, `CONFIG_KALLSYMS_ALL`, and
  `CONFIG_PROC_KCORE`, and `CONFIG_ISO9660_FS`

#### Scenario: Smoke uses non-vendored FreeBSD artifacts
- **WHEN** the FreeBSD kboot smoke stages `loader.kboot` and a FreeBSD or
  mfsBSD payload
- **THEN** those artifacts SHALL be downloaded or supplied by path outside the
  tracked repository and SHALL NOT be committed

#### Scenario: Smoke exposes payload through Linux hostfs
- **WHEN** the FreeBSD kboot smoke runs `loader.kboot` from Linux stage-1
- **THEN** it SHALL mount the FreeBSD or mfsBSD payload from Linux and run
  `loader.kboot` with `hostfs_root` and `bootdev=host:/` so unqualified
  `/proc` metadata reads resolve to the running Linux kernel while `/boot`
  reads resolve to the target payload

#### Scenario: Smoke preserves stock FreeBSD root media
- **WHEN** the FreeBSD kboot smoke uses a stock FreeBSD bootonly ISO
- **THEN** it SHALL expose the ISO as target-visible block media after the
  kernel jump, because the stock installer mounts `/` from its `cd9660` label
  and Linux-only hostfs or loop mounts do not survive as FreeBSD root devices

#### Scenario: Smoke can prove memory-root payloads without target media
- **WHEN** the FreeBSD kboot smoke uses an mfsBSD ISO or extracted mfsBSD
  payload root with a preloaded `mfsroot`
- **THEN** it SHALL support extracting or embedding that root tree into stage-1
  and omitting target-visible payload media, and success SHALL show the target
  mounting its md root or reaching an mfsBSD shell or login after the kernel
  jump

#### Scenario: Smoke forces target serial console output
- **WHEN** the FreeBSD kboot smoke runs `loader.kboot` from Linux stage-1
- **THEN** it SHALL pass FreeBSD kernel boot flags that enable serial and
  multiple consoles so installer or shell output is visible on the QEMU serial
  log after the kernel jump

#### Scenario: Smoke success signal reaches target environment
- **WHEN** the FreeBSD kboot handoff is reported as viable
- **THEN** the smoke evidence SHALL show serial output reaching the FreeBSD
  installer or an mfsBSD shell, not merely the FreeBSD loader menu

#### Scenario: Smoke failure keeps targets deferred
- **WHEN** the FreeBSD kboot smoke fails before reaching the FreeBSD installer
  or an mfsBSD shell
- **THEN** bootup SHALL keep targets for that failing route out of the
  executable default catalog and record the next blocking condition
