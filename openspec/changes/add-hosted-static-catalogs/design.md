## Context

Bootup currently accepts one static catalog source at startup: the embedded
default catalog or a local JSON file supplied with `--catalog`. That keeps the
stage-1 environment predictable, but it pushes hosted catalog update and
verification logic outside bootup. The existing static catalog model is already
data-only and provider-owned: catalog entries describe concrete targets for
compiled-in providers, while providers remain responsible for artifact
planning, staging, and boot-artifact verification.

The hosted-catalog slice should add remote document loading without changing
those boundaries. The remote document is still the same schema-versioned static
catalog JSON. It must be authenticated before parsing, checked for freshness,
and optionally cached so bootup can use a previously authenticated document
when a network source is unavailable.

## Goals / Non-Goals

**Goals:**

- Load static catalog JSON from an HTTPS URL when explicitly requested.
- Authenticate the downloaded bytes before parsing with either a SHA-256 digest
  pin or a detached Ed25519 signature and public key.
- Reject stale hosted catalogs using operator-provided freshness controls.
- Support an optional cache file that is updated only after authentication and
  catalog validation succeed.
- Support offline fallback to a cached hosted catalog when the network fetch
  fails and the cached document is still acceptable.
- Preserve existing embedded and local catalog behavior.

**Non-Goals:**

- Do not load provider code, policy scripts, dynamic discovery behavior, or
  runtime plugins from hosted catalogs.
- Do not change provider trust material for downloaded distro artifacts.
- Do not add a catalog merge model; a hosted catalog replaces the embedded
  catalog just like a local `--catalog` file does.
- Do not add certificate pinning or new third-party dependencies in this slice.

## Decisions

1. Add explicit hosted catalog flags instead of overloading `--catalog`.

   `--catalog` remains a local file path. Hosted loading uses
   `--catalog-url`, plus trust/freshness/cache flags. This avoids ambiguous
   path-vs-URL parsing and keeps existing automation stable.

2. Authenticate raw bytes before parsing JSON.

   Digest and signature checks run against the exact HTTP response body or
   cached bytes. Parsing happens only after authentication succeeds, which
   keeps malformed or tampered JSON from entering catalog validation paths.

3. Support two authentication modes.

   A SHA-256 digest pin is simple and good for immutable hosted catalog
   revisions. Ed25519 detached signatures are better for rolling catalog URLs.
   At least one mode is required for URL-hosted catalogs. If both are supplied,
   both must pass.

4. Carry freshness as a signed JSON field.

   The existing catalog schema uses `DisallowUnknownFields`, so add optional
   top-level `published_at` and `expires_at` fields to `catalog.Document`.
   `expires_at` is the primary freshness gate. `published_at` lets operators
   enforce a maximum catalog age when a catalog intentionally omits an expiry
   or uses a longer expiry than the local policy allows.

5. Make cache fallback explicit and conservative.

   A cache path is optional. Bootup writes it only after network bytes pass
   authentication, freshness checks, catalog parsing, and provider validation.
   If network fetching fails and `--catalog-cache-fallback` is set, bootup may
   load the cache after applying the same authentication, freshness, and
   catalog validation checks.

6. Keep HTTP behavior narrow.

   Hosted catalogs use HTTP GET with caller context, require `https://` URLs by
   default, reject non-2xx responses, and bound response size to avoid reading
   unbounded content into the initramfs environment. Tests can use HTTP servers
   by enabling an internal option; the CLI path requires HTTPS.

## Risks / Trade-offs

- Hosted catalogs can become an implicit trust root -> Require explicit digest
  or public-key trust configuration and document that provider artifact trust is
  still separate.
- Clock skew can break expiry checks -> Reuse existing time-preparation
  guidance and fail closed with clear errors when freshness cannot be proven.
- Cache fallback can mask upstream compromise or operator mistakes -> Recheck
  authentication and freshness on cached bytes, and make fallback opt-in.
- Long catalog files can exhaust memory in stage-1 -> enforce a bounded response
  size for hosted downloads and cached reads.
- Operators may expect hosted catalogs to merge with the default catalog ->
  keep replacement semantics consistent with local `--catalog` and document it.

## Migration Plan

Existing invocations without hosted flags continue using the embedded catalog.
Existing `--catalog=/path/to/catalog.json` invocations continue loading local
files. Operators can migrate by publishing the same JSON document at an HTTPS
URL, signing it or pinning its SHA-256 digest, and passing `--catalog-url` with
the matching trust flags. Rollback is removing hosted flags or pointing
`--catalog` at a local copy of the catalog.

## Open Questions

- Whether a future release should support multiple catalog URLs with priority
  ordering or fallback mirrors.
- Whether signature metadata should eventually move into a sidecar manifest
  format when catalog publication needs multiple signatures or key rotation.
