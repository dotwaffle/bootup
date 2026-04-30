## Context

Bootup has three intended provider/catalog modes:

1. Static concrete targets from embedded or local static catalogs.
2. Compiled-in provider logic that dynamically discovers concrete targets when
   an operator selects a distro family.
3. Fully dynamic policy or script/server-driven boot decisions.

Mode 2 should bridge the gap between a maintained static catalog and fully
dynamic policy. It must stay within compiled-in Go provider code and explicit
runtime configuration, because stage-1 environments need predictable behavior
and clear trust boundaries.

## Goals / Non-Goals

**Goals:**

- Define a provider-facing discovery model for distro families and discovered
  concrete targets.
- Support discovery of releases, architectures, variants, and install options.
- Allow optional lifecycle decoration, such as supported/obsolete/EOL, without
  making lifecycle data mandatory for booting.
- Keep discovered targets compatible with the existing boot planning and staging
  path.
- Preserve static catalogs as the simple mode-1 path.

**Non-Goals:**

- Loading provider code from remote catalogs or runtime plugins.
- Hosted static catalog URL fetching.
- Script execution or self-hosted policy decisions.
- Requiring public Internet access for all providers.
- Replacing the existing static catalog.

## Decisions

### Introduce provider families before discovered targets

The operator should first see static concrete targets and/or provider families.
A provider family is a selectable discovery entry such as `debian` or `ubuntu`
that describes what the provider can discover. Selecting a family runs compiled
provider logic and returns concrete targets using the same `provider.Target`
shape used by static catalogs.

Alternatives considered:

- Discover everything at startup: simple for UI but slow and fragile when
  network or remote metadata is unavailable.
- Make static catalog entries contain discovery scripts: flexible, but crosses
  into mode-3 policy/plugin behavior.

### Discovery results are data, planning remains provider code

Discovery returns concrete targets plus source/catalog metadata. The provider's
existing plan/stage methods still own artifact URL construction, verification
material interpretation, and handoff behavior. This keeps discovered targets
compatible with the current app flow.

### Lifecycle metadata is optional decoration

Providers can attach lifecycle status such as `supported`, `obsolete`, `eol`,
or `unknown`, along with a source string and optional date. The UI can display
this information, but booting a target should not depend on lifecycle metadata
unless a future operator policy feature makes that explicit.

### Runtime configuration supplies discovery sources

Providers can use compiled-in defaults, provider runtime config, or local files
to discover releases. External lifecycle services such as endoflife.date should
be optional and timeout-bound. Providers must report partial discovery failures
clearly instead of silently returning misleading target lists.

## Risks / Trade-offs

- Discovery can be slow or flaky → Run it only after selecting a family, use
  explicit timeouts, and render clear progress/failure messages.
- Discovered metadata can be stale or incomplete → Treat it as provider data and
  preserve static catalog fallback paths.
- Lifecycle decoration can be mistaken for a trust signal → Keep lifecycle
  status separate from verification and document that it is informational.
- Provider family UX can complicate menus → Start with a simple two-step flow:
  choose family, then choose discovered target.

## Migration Plan

Implement mode 2 as an additive provider interface. Existing providers and
static catalog targets continue to work unchanged. UIs can render discovery
families only when providers implement the new interface.

## Open Questions

- Should discovered targets be cached during one boot session, and if so, for
  how long?
- Should discovery support local metadata files before remote discovery?
- How should operators configure lifecycle metadata sources and timeouts?
- Should static targets and discovered family entries be rendered in the same
  list or separate tabs/sections in rich UI?
