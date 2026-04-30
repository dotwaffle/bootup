## Context

The current discovery implementation proves the provider-family model with
Debian amd64 netboot discovery. The next slice should make the model less
provider-specific while preserving the existing trust boundaries:

- Provider code remains compiled into bootup.
- Static catalogs stay static data and are not executable discovery logic.
- Lifecycle metadata remains decoration, not verification material.
- Hosted catalog URL fetching remains a future capability.

## Design

### Provider Runtime Config

Add provider-specific optional fields:

- `discovery_url`: HTTP(S) root or index URL used by that provider's discovery
  implementation.
- `discovery_timeout`: Go duration string such as `2s` or `500ms`.
- `lifecycle`: map keyed by provider release string. Values reuse the target
  lifecycle shape: status, source, and optional date.

Debian defaults `discovery_url` to its configured mirror URL. Ubuntu defaults
`discovery_url` to the Ubuntu releases index. Empty discovery timeouts use each
provider's default. Runtime config loading validates URLs, duration strings,
lifecycle status values, and lifecycle dates before provider registration.

### Ubuntu Discovery

Ubuntu discovery fetches the configured releases index, extracts release links,
fetches each release's `SHA256SUMS`, and selects releases that include an
`ubuntu-<version>-live-server-amd64.iso` entry. It then probes the release's
`netboot/amd64/linux` and `netboot/amd64/initrd` paths. Successful probes
return concrete `provider.Target` values with `source.base_url` and
`source.iso_name` so the existing Ubuntu planner can resolve artifacts.

Discovery does not stage or verify boot artifacts. It only discovers concrete
targets. Planning, HTTPS constraints, optional signed checksum validation, and
artifact staging remain provider-owned.

### Lifecycle Decoration

Providers attach configured lifecycle entries to matching discovered releases.
When no lifecycle entry exists, providers may return `unknown` lifecycle
decoration. The UI renders lifecycle decoration as text only; verification code
does not inspect lifecycle fields.

### UX And Failure Handling

`list-targets` continues to list static catalog targets only. Discovery runs
only when a menu family is selected or `discover-targets` mode is used.
Discovery diagnostics report empty results with a clear message instead of a
generic error. Discovery failures do not alter loaded static catalog targets.

### Hosted Static Catalogs

This change documents the required future design: URL catalogs need explicit
authenticity and freshness semantics before runtime loading. Candidate models
include signed catalog documents or pinned digest/cosign verification, with
operator-controlled cache and offline fallback behavior. Runtime URL catalog
loading remains unimplemented.

## Risks / Trade-offs

- HTML directory indexes vary by mirror. Keep parsers narrow and test with
  fixture data that resembles official indexes.
- Discovery can be slow or partial. Keep it explicit, timeout-bound, and
  provider-scoped.
- Lifecycle data can be mistaken for trust. Keep it in `Target.Lifecycle` and
  do not pass it into verification APIs.
- Adding too many static targets can create broken choices. Add only targets
  whose provider planning shape is already supported.
