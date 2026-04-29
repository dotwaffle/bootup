## Context

The archived MVP established the stage-1 environment, Debian provider,
verification hooks, and kexec executor. The default build still has no Debian
archive keyring, by design. A local builder can generate ignored Go source from
a chosen OpenPGP keyring, yielding a single binary with explicit trust material.

This change proves that path end to end and improves the interface and provider
metadata needed before adding more operating systems.

## Decisions

### Keep real Debian smoke opt-in

Live Debian smoke tests require network, QEMU, a local kernel, and local trust
material. They will not run in default CI. The test should require explicit
environment variables and skip otherwise.

### Keep keyring generation outside repository history

The real smoke path uses `cmd/bootup-keyring-source` to generate ignored source.
No Debian keyring, generated keyring source, initramfs, or binary output should
be committed.

### Improve the text UI before framebuffer work

The serial UI should support index selection, clear progress/status lines, and
fatal errors that remain readable in an 80x25 viewport. This remains the
baseline even when a framebuffer UI is added later.

### Add catalog metadata without adding a catalog engine

Provider targets should expose stable fields for distribution, release,
architecture, and kind. The current UI can render a simple grouped list, while
future work can add nested catalog navigation.

## Risks / Trade-offs

- Live Debian mirrors may be temporarily unavailable, so real smoke must be
  opt-in and diagnostically clear.
- QEMU kexec behavior can differ from host execution; local host smoke may
  stage successfully but fail kexec with `EPERM`.
- Catalog metadata should not overfit Debian; keep field names generic and
  optional.

## Rollout

1. Add repeatable local smoke commands and opt-in smoke scripts/tests.
2. Improve serial menu rendering and selection states.
3. Add provider catalog metadata and update Debian target data.
4. Run normal and fixture-tag checks, then run real smoke where local inputs
   are present.
