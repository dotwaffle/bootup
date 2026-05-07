## Why

Operators can already replace bootup's embedded catalog with a local JSON file,
but hosted catalog content still has to be fetched and verified outside bootup.
Bootup should support URL-hosted static catalogs directly while preserving the
current trust boundary: catalogs are signed or pinned data, not remote code.

## What Changes

- Add URL-hosted static catalog loading as an explicit catalog source mode.
- Require hosted catalogs to be authenticated with either a caller-supplied
  SHA-256 digest pin or detached Ed25519 signature and public key.
- Add freshness controls for hosted catalogs so operators can reject stale
  documents before provider registration.
- Add an optional on-disk cache with offline fallback when a previously
  authenticated catalog remains fresh enough for use.
- Keep static catalog documents non-executable: hosted catalogs can describe
  concrete targets for compiled-in providers but cannot load provider code,
  execute scripts, or perform dynamic discovery.
- Document catalog authenticity separately from provider artifact
  verification. Distribution boot artifacts still use provider-owned trust
  material and checksum/signature checks.

## Capabilities

### New Capabilities
- `bootup-hosted-static-catalogs`: URL loading, authentication, freshness, and
  cache behavior for hosted static catalog documents.

### Modified Capabilities
- `bootup-static-catalog-source`: static catalog sources include embedded,
  local-file, and authenticated URL-hosted documents while retaining the same
  validated document model.
- `bootup-static-provider-catalog`: provider target registration must treat
  hosted catalog targets the same as other validated static catalog targets and
  reject unsupported provider/action/data shapes before rendering or planning.

## Impact

- Affects `cmd/bootup` catalog source flags and startup loading.
- Adds hosted catalog fetch/authentication/cache logic under `internal/catalog`.
- Updates catalog tests, command startup tests, operator documentation, and
  OpenSpec requirements.
- Uses only Go standard library cryptography and HTTP packages; no new runtime
  dependencies are expected.
