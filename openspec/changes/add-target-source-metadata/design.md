## Context

Bootup's static catalog currently records identity and grouping metadata for
concrete targets. Debian URLs can be derived from release and architecture
against a provider mirror, but Ubuntu live-server netboot needs two concrete
target facts: the release directory URL and the live-server ISO name used on
the kernel command line and in signed checksum validation.

The catalog should remain data only. Provider code stays compiled in, provider
runtime config continues to supply trust material and coarse provider defaults,
and hosted/dynamic catalog behavior stays out of scope.

## Goals / Non-Goals

**Goals:**

- Add optional per-target source metadata without changing the catalog schema
  version.
- Keep existing catalog documents valid when they omit source metadata.
- Preserve existing provider runtime configuration for the current default
  targets.
- Allow Ubuntu catalog entries to define per-target release URLs and installer
  ISO names.
- Expand the embedded default catalog with targets that the provider logic can
  plan honestly.

**Non-Goals:**

- Per-target trust material or checksum pins.
- Hosted catalog URL loading.
- Dynamic release discovery.
- Non-amd64 target support.
- Open-ended provider-specific JSON blobs in the catalog.

## Decisions

### Add a small typed source block

`provider.Target` will gain optional source metadata:

- `source.base_url`: an HTTP(S) URL that acts as the selected target's provider
  source root.
- `source.iso_name`: a pathless ISO filename used by providers that boot a
  netboot kernel into a live/server ISO.

This keeps the schema readable and avoids a provider-specific untyped map. It
is intentionally small; additional source fields should be added only when a
provider needs them.

Alternatives considered:

- Provider-specific `source` objects: flexible, but weakens common validation
  and makes the catalog harder to inspect.
- More provider-level runtime config: preserves the current shape, but cannot
  represent multiple Ubuntu releases in one static target list.

### Source precedence is target first when present

Provider planning will use `target.Source.BaseURL` when present. When it is
absent, the provider keeps its existing provider runtime/default source. This
preserves current behavior for the original Ubuntu 26.04 default target while
allowing new sourceful targets such as Ubuntu 24.04.4 and 25.10.

For Debian, source metadata can override the mirror for a target, but the
default catalog can continue to omit it because the provider mirror applies to
all listed Debian releases.

### Keep verification config provider-level for now

The existing runtime config can still provide Debian keyrings and Ubuntu release
keyrings or netboot artifact hash pins. Per-target trust material is not part
of this change. If operators need per-target Ubuntu hash pins later, that should
be a separate runtime config or catalog trust design.

## Risks / Trade-offs

- Provider-level Ubuntu hash pins apply to any selected Ubuntu target → Keep
  current fail-closed behavior; mismatched pins fail artifact verification.
- Ubuntu point-release ISO names can become stale → Static catalog updates are
  expected to change concrete target facts; this is mode-1 behavior.
- Generic source fields may not cover future providers → Add fields only when
  a provider demonstrates a real need, rather than accepting arbitrary blobs.
- Runtime provider source overrides cannot override sourceful targets → Use a
  local catalog replacement to change per-target source URLs until a richer
  mirror/override model is designed.

## Migration Plan

Existing catalogs without `source` remain valid. Existing 26.04 Ubuntu behavior
continues to use the provider runtime/default release URL when the default
catalog target omits `source.base_url`. New default targets carry source data
where required for correct planning.
