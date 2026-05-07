## Context

Debian, Ubuntu, and Fedora discovery fetch directory-style indexes and then
probe per-release metadata/artifact paths. The configured source URLs are also
used later for artifact planning and staging, so local discovery fixtures need
to override only the metadata read path. The shared HTTP helpers already
centralize fetch/probe behavior, making them the right boundary for local file
metadata.

## Goals / Non-Goals

**Goals:**

- Allow hermetic discovery fixtures through local `discovery_file` paths.
- Keep provider artifact/release source URLs HTTP(S)-only unless separately
  designed.
- Continue discovery when one candidate release probe fails.

**Non-Goals:**

- Add a new catalog format for discovery results.
- Load provider code, policy, or scripts from local files.
- Treat local metadata as artifact trust material.

## Decisions

- Add `discovery_file` beside existing `discovery_url`. The file path selects
  where discovery metadata is read from, while the existing source URL still
  selects the HTTP(S) base URL that discovered targets use for planning and
  staging.
- Add `file://` support to shared fetch/probe helpers. Directory fetches read
  `index.html`; file probes use `os.Stat`; missing paths map to 404/absence.
- Validate `discovery_file` as a local filesystem path and convert it to an
  internal `file://` metadata URL at the provider boundary.
- Keep primary index failures fatal. Per-release probe failures are treated as
  candidate absence so one bad release does not hide other valid targets.

## Risks / Trade-offs

- File discovery metadata can become stale -> document it as an operator-owned
  discovery source, not trust material.
- Skipping probe errors can hide a broken release candidate -> discovery still
  returns no targets when every candidate fails, and diagnostics can capture the
  run context if needed.
