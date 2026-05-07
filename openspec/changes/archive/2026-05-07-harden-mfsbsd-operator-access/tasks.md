## 1. FreeBSD Kboot Option Flow

- [x] 1.1 Add regression coverage for applying selected options to `freebsd-kboot` loader arguments.
- [x] 1.2 Preserve pre-stage `freebsd-kboot` loader option arguments when staging generates default payload-root arguments.

## 2. mfsBSD Runtime Defaults

- [x] 2.1 Pass explicit `mfsbsd.autodhcp=YES` and default hostname loader variables for mfsBSD targets.
- [x] 2.2 Add an mfsBSD hostname target option to the default provider/catalog data and regenerate embedded catalog output.

## 3. Documentation and Verification

- [x] 3.1 Document mfsBSD login, serial console, DHCP, SSH, packages, and hostname option behavior.
- [x] 3.2 Run package tests, generated catalog checks, OpenSpec validation, and repository verification for the changed scope.
