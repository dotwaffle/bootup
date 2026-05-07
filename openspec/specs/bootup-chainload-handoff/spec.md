# bootup-chainload-handoff Specification

## Purpose
Define the evidence and artifact handling requirements for bootloader,
firmware, memdisk, COM32, HDT, OpenBSD, and stock BSD installer handoff routes
before bootup exposes them as executable catalog targets.

## Requirements
### Requirement: Chainload viability is evidence-based
Bootup SHALL treat bootloader, firmware, memdisk, COM32, HDT, OpenBSD, and
stock BSD installer chainload support as deferred until a reproducible smoke
proves a candidate handoff reaches the target environment.

#### Scenario: Candidate route is assessed
- **WHEN** bootup evaluates a chainload candidate route
- **THEN** it SHALL record the bootloader or firmware mechanism, source
  artifacts, staged media layout, target-visible devices or memory payloads,
  firmware assumptions, and exact QEMU command or helper used by the test

#### Scenario: Viability claim reaches target environment
- **WHEN** a chainload route is reported as viable
- **THEN** the evidence SHALL include serial or console output from the target
  environment after the chainloaded kernel, installer, tool, or bootloader has
  started, not merely evidence that bootup launched a helper process

#### Scenario: Blocker keeps route deferred
- **WHEN** a candidate route cannot acquire a loader, preserve required
  firmware state, expose media after handoff, or reach the target environment
- **THEN** bootup SHALL keep targets for that route out of the executable
  default catalog and document the blocking boundary

### Requirement: Chainload proof artifacts are not vendored
Bootup SHALL NOT commit generated bootloader binaries, distro payloads, ISO
images, disk images, firmware variable images, or generated initramfs images as
part of a chainload investigation.

#### Scenario: Experimental materials are generated locally
- **WHEN** a chainload proof requires bootloader, firmware, kernel, installer,
  disk, ISO, or memory payload artifacts
- **THEN** those artifacts SHALL be downloaded, copied, or built into ignored
  local paths or `/tmp`, with enough provenance recorded to reproduce them

### Requirement: Chainload route explains target-visible media
Bootup SHALL require each chainload candidate to describe how the target
environment sees its required boot or root media after control leaves Linux
stage-1.

#### Scenario: Target needs persistent media
- **WHEN** the target kernel, installer, or tool expects a disk, ISO, RAM disk,
  network boot resource, or firmware-loaded image after handoff
- **THEN** the candidate SHALL identify whether that resource is provided by
  real hardware, firmware, a target-native RAM payload, externally attached
  QEMU media, or another mechanism that survives Linux process exit and kernel
  replacement

#### Scenario: Linux-only media is insufficient
- **WHEN** the candidate only exposes required media through Linux hostfs,
  loopback, tmpfs, or a process-local file descriptor
- **THEN** the route SHALL remain a loader-only proof unless target boot
  evidence shows the media remains usable after handoff
