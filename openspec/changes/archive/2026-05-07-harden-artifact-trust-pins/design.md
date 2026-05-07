## Context

The generic Linux provider stages kernel/initrd artifacts from catalog source
metadata and already verifies an artifact hash if `provider.Artifact.SHA256` is
set. The missing link is catalog data: `provider.SourceEntry` can describe
paths but not artifact hashes, so hosted or local catalog operators cannot make
generic Linux targets hash-pinned without provider code changes.

## Goals / Non-Goals

**Goals:**
- Add data-only SHA-256 pins for generic Linux catalog kernel/initrd artifacts.
- Validate malformed or partial source pins before provider registration.
- Preserve pins through generated static catalog output.
- Reuse existing provider staging verification and catalog matrix
  classification.

**Non-Goals:**
- Do not add default catalog pins that require live mirror hash research in
  this change.
- Do not add OpenPGP signing policy for generic Linux mirrors.
- Do not change Debian, Ubuntu, Fedora, or mfsBSD provider trust behavior.

## Decisions

- Extend `provider.SourceEntry` with `KernelSHA256` and `InitrdSHA256`.
  This keeps hash pins with the source paths they verify and lets local,
  hosted, and generated catalogs use the same schema.

- Validate source pins centrally in provider target validation. Hash values
  must be lowercase-normalized by planning and must be valid 64-character
  SHA-256 hex digests. If an initrd path is present and either kernel or initrd
  pin is supplied, both pins are required so the target does not appear
  partially hardened.

- Keep default catalog source entries unpinned for now. Adding pins for rolling
  or mirror-served installer artifacts should happen only when the source has a
  stable, maintainable pin update workflow.

## Risks / Trade-offs

- Existing external catalogs with malformed new fields will start failing.
  Mitigation: the fields are new and optional; only catalogs that opt into them
  are affected.

- Hash pins can drift when upstream mirrors update rolling artifacts.
  Mitigation: leave default rolling targets unpinned and document that operators
  should pin only artifacts they can update deliberately.

- Central source validation cannot know every provider's semantic use of source
  fields. Mitigation: this change only makes the generic Linux provider consume
  these fields, and the matrix reflects pins only after the provider includes
  them in the boot plan.
