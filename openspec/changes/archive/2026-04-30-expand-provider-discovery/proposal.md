## Why

Bootup has the first dynamic discovery path, but it is still Debian-only and
has a few rough edges: stale docs still describe discovery as deferred, Ubuntu
cannot discover release targets dynamically, discovery source and timeout
configuration are provider-internal, lifecycle decoration cannot be supplied by
operators, and hosted catalog authenticity is only loosely described.

## What Changes

- Refresh docs/spec wording so dynamic provider discovery is treated as an
  implemented mode.
- Add configurable discovery source URLs, discovery timeouts, and lifecycle
  decoration maps to provider runtime config.
- Add Ubuntu amd64 netboot discovery from the Ubuntu releases index.
- Keep lifecycle metadata informational and separate from verification/trust.
- Improve discovery empty/failure messages without changing static target
  listing semantics.
- Add one broader static catalog entry for Debian forky amd64 netboot.
- Document the hosted static catalog authenticity/freshness design boundary
  without implementing URL catalog loading.

## Impact

- Affects provider runtime config parsing, Debian and Ubuntu provider
  construction, app/UI discovery diagnostics, docs, tests, and OpenSpec specs.
- Does not add hosted catalog URL fetching, remote provider plugins, dynamic
  policy scripts, distribution keyrings, release tagging, or SLSA release work.
