## Context

Bootup already has three static catalog sources: embedded default, local file,
and authenticated hosted URL. Local and hosted sources are replacements. That is
safe but inconvenient for operators who want to add a small number of private
or lab targets while keeping upstream defaults available.

## Goals / Non-Goals

**Goals:**

- Let operators include the embedded default catalog alongside one selected
  local or hosted catalog.
- Preserve replacement as the default behavior for compatibility.
- Keep target ID conflict behavior simple and deterministic.

**Non-Goals:**

- Multiple local catalog paths or multiple hosted URLs.
- Target override, shadowing, deletion, or patch semantics.
- New catalog schema fields.
- Composing unauthenticated hosted catalog bytes.

## Decisions

- Add `--catalog-include-default` as an explicit boolean flag. This makes
  composition visible at the CLI and avoids changing current replacement
  semantics.
- Compose after both inputs have been parsed and validated. Hosted catalogs
  still pass authentication, freshness, size, cache, and schema validation
  before their targets can be combined with embedded defaults.
- Add a catalog package helper that builds a new `Document` from validated
  documents and rejects duplicate target IDs. The helper preserves target order
  by appending sources in caller order.
- Reject duplicate IDs rather than overriding. Full replacement remains the
  operator path for deliberate overrides, and unique target IDs keep menu,
  plan, stage, and diagnostics behavior predictable.

## Risks / Trade-offs

- Operators may expect override behavior -> document that composition is
  additive only and duplicate IDs are invalid.
- The composed document has multiple provenance roots -> keep freshness metadata
  from the selected hosted/local document out of the composed result and use the
  existing catalog source posture to describe how the selected catalog was
  loaded.
- Larger target lists can make menu output longer -> composition is opt-in.
