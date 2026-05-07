## Why

Dynamic discovery currently depends on live HTTP directory indexes and aborts
when one candidate release has a transient probe failure. Operators need a
hermetic way to test or seed discovery from local metadata, and discovery
should keep useful candidates when one release path is broken.

## What Changes

- Add provider `discovery_file` settings that point discovery at local
  metadata fixtures while release/artifact source URLs remain HTTP(S).
- Teach shared provider HTTP helpers to fetch and probe local `file://` URLs
  for discovery fixtures and offline operator workflows.
- Skip individual discovery candidates whose optional probe metadata fails,
  while still failing when the primary discovery index cannot be fetched or
  parsed.
- Add provider, config, shared helper, and CLI tests plus docs for local
  metadata discovery.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `bootup-dynamic-distro-discovery`: discovery supports local file metadata
  paths and continues past per-candidate probe failures.
- `bootup-provider-runtime-config`: provider runtime config accepts
  `discovery_file` paths for local discovery metadata.

## Impact

- Affected code: `internal/providerhttp`, `internal/providerconfig`,
  Debian/Ubuntu/Fedora discovery loops, docs, and tests.
- Public behavior: provider config may point `discovery_file` at local metadata.
- Dependencies: no new third-party dependencies.
