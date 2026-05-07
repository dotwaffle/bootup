## Context

The catalog matrix is a dry-run view over registered targets. It currently
renders target ID, provider, action, plan status, artifact trust, smoke
coverage, and error text. Providers are gaining normal planning paths that may
fetch remote metadata, so the matrix needs an explicit way to request offline
planning while keeping the operator output richer.

## Goals / Non-Goals

**Goals:**

- Keep `catalog-matrix` hermetic even when normal target planning can fetch
  metadata.
- Include enough catalog identity fields in each row to read the matrix without
  cross-referencing `list-targets`.
- Preserve a simple TSV output shape.

**Non-Goals:**

- Add JSON output or alternate render formats.
- Add new smoke helpers or change coverage classifications.
- Change normal plan/stage/boot target behavior.

## Decisions

- Add an `Offline` boolean to `provider.PlanInput`. The zero value preserves
  normal provider planning. `catalog.BuildConformanceReport` sets `Offline:
  true` so providers can fail deterministic dry-run planning instead of
  fetching remote metadata.
- Implement the offline check only where it matters now: Fedora `.treeinfo`
  fallback. Existing providers either plan from local metadata or already avoid
  network work in planning.
- Expand the TSV row rather than adding a separate summary block. Additional
  columns keep the output machine-readable with one row per target.
- Render lifecycle as its status string and leave it blank when unset. The
  lifecycle source/date remain detailed target metadata for `show-target`.

## Risks / Trade-offs

- Additional TSV columns can affect ad hoc parsers -> update docs and tests,
  and keep columns stable once introduced.
- Offline planning can surface errors for custom unpinned Fedora targets in the
  matrix even though normal planning could fetch `.treeinfo` -> error text
  explains the missing offline pins and preserves the matrix no-network
  contract.
