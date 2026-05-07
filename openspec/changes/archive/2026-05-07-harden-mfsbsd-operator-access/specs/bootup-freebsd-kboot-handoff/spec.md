## MODIFIED Requirements

### Requirement: mfsBSD kboot target stages verified runtime artifacts
Bootup SHALL stage executable mfsBSD `freebsd-kboot` targets from verified
runtime downloads and SHALL present the extracted mfsBSD root tree through
Linux hostfs.

#### Scenario: mfsBSD target stages loader and memory-root payload
- **WHEN** bootup stages an mfsBSD `freebsd-kboot` target
- **THEN** it SHALL verify a pinned mfsBSD ISO hash, extract the ISO contents
  from Linux without requiring the Linux kernel to mount the ISO, normalize
  compressed `kernel` and `mfsroot` payload files when needed, verify a pinned
  FreeBSD base archive hash, extract `loader.kboot` and `loader.help.kboot`,
  and prepare loader arguments containing `hostfs_root`, `bootdev=host:/`,
  serial console settings, `mfsbsd.autodhcp=YES`, and an mfsBSD hostname
  setting

#### Scenario: mfsBSD selected loader options survive staging
- **WHEN** an operator selects a non-secret mfsBSD target option
- **THEN** bootup SHALL preserve the selected option as a `loader.kboot`
  argument after generated default loader arguments are prepared during staging
