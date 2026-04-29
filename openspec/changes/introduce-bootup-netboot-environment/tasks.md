## 1. Project Skeleton

- [x] 1.1 Create the bootup command package and startup flow for the stage-1 environment
- [x] 1.2 Add a provider registry that exposes build-time provider modules
- [x] 1.3 Define boot target, boot plan, artifact, verification, and handoff types
- [x] 1.4 Add structured logging suitable for serial console diagnostics

## 2. Runtime Environment

- [x] 2.1 Build a u-root initramfs image that includes bootup and required base tools
- [x] 2.2 Add network setup for DHCP and DNS before provider operations
- [x] 2.3 Add CA certificate material through binary-embedded TLS roots
- [x] 2.4 Add time sanity checking and a configured time synchronization path

## 3. Operator Interface

- [x] 3.1 Implement a text-mode menu that works in an 80x25 serial viewport
- [x] 3.2 Render provider targets, progress, and fatal errors through the text interface
- [x] 3.3 Add a non-interactive selection mode for integration tests

## 4. Debian Provider

- [x] 4.1 Implement Debian trixie amd64 target discovery
- [x] 4.2 Resolve Debian Installer netboot kernel, initrd, metadata, and checksum URLs
- [x] 4.3 Add a compile-time Debian archive trust-material hook without committing keyrings
- [x] 4.4 Verify signed Debian archive metadata before trusting installer checksums
- [x] 4.5 Download and verify the selected Debian Installer kernel and initrd
- [x] 4.6 Return a complete boot plan with kernel path, initrd path, and command line

## 5. Kexec Handoff

- [x] 5.1 Choose and document the MVP kexec implementation path
- [x] 5.2 Stage verified kernel and initrd artifacts for kexec
- [x] 5.3 Execute kexec with the provider command line
- [x] 5.4 Detect and report kexec failures without losing diagnostics

## 6. Launch Artifacts

- [x] 6.1 Add a QEMU/vmtest launch path for the bootup kernel and initramfs
- [x] 6.2 Add an iPXE chainload example for bootup
- [x] 6.3 Add a GRUB menu entry example for bootup
- [x] 6.4 Add notes for ISO-based bootup delivery

## 7. Verification

- [x] 7.1 Add unit tests for provider registration and boot planning
- [x] 7.2 Add unit tests for Debian metadata and checksum verification failures
- [x] 7.3 Add vmtest coverage that boots bootup to the text interface
- [x] 7.4 Add vmtest coverage for Debian artifact resolution, verification, and staging
- [x] 7.5 Document Secure Boot and kernel lockdown failure modes discovered during testing
