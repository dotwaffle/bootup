## Context

Bootup currently exposes static provider targets from provider constructors.
That works for a small MVP, but it puts mode-1 catalog expansion in Go source
and makes operator-supplied static catalogs awkward. The provider code itself
must remain compiled in; a catalog document should describe concrete targets,
not load runtime provider plugins.

The operator's long-term model has three layers:

1. Static concrete boot targets.
2. Static provider logic that can dynamically discover releases, arches, and
   options.
3. Fully dynamic policy or script-driven boot decisions.

This change implements the data source for layer 1 and documents the boundary
before layer 2.

## Goals / Non-Goals

**Goals:**

- Source concrete static targets from an embedded JSON catalog.
- Allow an operator-provided local JSON catalog to replace the embedded
  catalog at startup.
- Validate catalog documents before target discovery.
- Keep provider code compiled in and explicitly wired.
- Expand the default catalog with Debian bookworm amd64 netboot.
- Document hosted catalogs and dynamic discovery as future work.

**Non-Goals:**

- Loading provider code from catalog documents.
- Fetching catalog documents from URLs at runtime.
- Implementing dynamic release or architecture discovery.
- Adding distro-specific trust bundles to release artifacts.
- Adding per-target Ubuntu release URL semantics.

## Decisions

### Use a versioned JSON document

The catalog format will start as JSON with `schema_version: 1` and a `targets`
array whose entries match `provider.Target` plus typed catalog metadata. JSON
keeps the first implementation stdlib-only and easy to embed, validate, and
override in initramfs environments.

Alternatives considered:

- YAML/TOML/HCL: friendlier for hand editing, but adds dependencies or parser
  choices before the schema has stabilized.
- Go source only: simplest today, but does not provide the operator-supplied
  static catalog path.

### Treat `--catalog` as a replacement, not a merge

When `--catalog` is supplied, bootup will load that local file instead of the
embedded default. Replacement behavior avoids surprising precedence rules and
keeps validation deterministic.

Alternatives considered:

- Merge external and embedded targets: useful later, but it needs explicit
  duplicate and override semantics.
- Provider-specific catalog flags: less useful for the mode-1 catalog, which is
  intended to describe the whole operator-facing list.

### Validate before provider registration

Catalog validation will reject malformed JSON, unsupported schema versions,
duplicate target IDs, unknown provider IDs, provider ID mismatches, missing
display names, and incomplete catalog metadata before providers are registered.
Providers may still validate their own release or architecture constraints
during planning.

### Pass filtered targets into compiled-in providers

The command wiring will filter catalog targets by provider ID and pass those
slices into compiled-in providers. Providers remain responsible for planning
and staging, but the static target list becomes data-driven.

For Debian, plan URLs can be derived from release and architecture metadata for
the existing amd64 netboot path. For Ubuntu, the default catalog remains one
26.04 target because the provider currently uses a provider-level release URL.
Additional Ubuntu releases should wait for explicit per-target source URL
semantics.

### Defer hosted catalogs and dynamic discovery

Hosted catalog URLs raise questions about authenticity, freshness, pinning,
cache behavior, and offline fallback. Dynamic discovery raises a different set
of requirements around provider discovery contracts, end-of-life metadata, and
user interaction. This change documents both directions but only implements the
local/static catalog source.

## Risks / Trade-offs

- Local catalog files can describe provider targets the provider cannot plan →
  Validate generic metadata at startup and fail closed during provider planning
  for unsupported provider-specific values.
- JSON is less ergonomic than YAML for hand-written catalogs → Keep the schema
  small and revisit format options after the static catalog grows.
- Replacement-only overrides make it harder to add one target to defaults →
  Favor predictable semantics now; add merge behavior later only with clear
  precedence rules.
- Debian release paths could drift between releases → Restrict the initial
  generalized provider path to amd64 Debian Installer netboot paths covered by
  tests.

## Migration Plan

Existing users continue to get the embedded default catalog without changing
flags. Operators that want a fully custom static list can pass `--catalog` with
a local JSON document. If a custom catalog fails validation, bootup fails before
provider discovery rather than falling back silently.

## Open Questions

- What authenticity model should a hosted catalog URL use: signed catalog,
  pinned HTTPS origin, content hash pin, cosign bundle, or some combination?
- Should future catalog merge behavior allow additive local entries, overrides,
  removals, or all three?
- What schema should mode-2 dynamic discovery use to describe release,
  architecture, install-option, and EOL metadata?
