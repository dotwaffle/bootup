## Context

The current provider API exposes concrete boot targets directly. Each target
already carries distribution, release, architecture, and kind strings, and the
text/rich UIs use those strings for grouping. That is enough for two providers,
but it leaves the catalog contract implicit as bootup grows toward a large
static list of known bootable distro targets.

The long-term model has at least three modes:

- Static concrete targets embedded in the tool or loaded from static catalog
  content.
- Static distro/provider entries that discover currently available releases,
  architectures, and options dynamically.
- Fully dynamic policy or script-driven boot decisions.

This change only implements the first mode's target metadata foundation.

## Goals / Non-Goals

**Goals:**

- Make static concrete target catalog metadata explicit and typed.
- Ensure registered provider targets carry catalog metadata before the UI sees
  them.
- Keep existing Debian/Ubuntu behavior and target IDs stable.
- Keep current list/menu presentation behavior stable while moving it to the
  catalog field.

**Non-Goals:**

- No external hosted catalog file yet.
- No YAML/JSON/HCL catalog parser yet.
- No dynamic distro release discovery.
- No end-of-life metadata or remote decoration.
- No script/plugin/server policy execution.
- No UI filtering/search redesign.

## Decisions

### Embed typed catalog metadata in `provider.Target`

Add `provider.CatalogEntry` and a `Catalog provider.CatalogEntry` field to
`provider.Target`. Catalog metadata belongs with the concrete boot target
because target selection, grouping, and planning all use the same identity.

Alternative: add a separate catalog registry alongside provider targets. That
will become useful when external static catalogs or discovery providers exist,
but it would duplicate state before there is a second data source.

### Require non-empty catalog metadata from registered providers

`Registry.Targets` will validate every provider target before returning it.
Each target must have ID, provider ID, display name, and non-empty catalog
metadata with distribution, release, architecture, and kind for current static
distro targets. Provider ID must match the provider that returned it.

Alternative: leave validation to UI tests. That keeps registration permissive
but allows broken providers to reach operator-facing flows.

### Keep UI grouping as a consumer of catalog metadata

Text and rich menus will render from `target.Catalog`. They will preserve the
existing display format, so this change is a data-contract change rather than a
UI feature.

Alternative: add filtering/search now. That is better saved for a larger
catalog, when the real operator workflow is visible.

## Risks / Trade-offs

- Stricter provider validation can break tests and future non-distro targets ->
  update current tests explicitly and revisit validation when non-distro target
  kinds are introduced.
- The static catalog type may need more fields later -> keep this slice focused
  on stable grouping fields and add lifecycle/options/source metadata in future
  changes.
- Dynamic discovery and fully dynamic policy are intentionally deferred -> name
  them in docs so the static model does not imply they are rejected.
