## Why

Signed local policy decisions now provide a narrow data-only automation path,
but the workflow is still awkward to operate. Operators need a repeatable way
to author and sign policy files, a hosted policy source with the same
fail-closed trust posture as local policy, smoke coverage for the full signed
decision path, and an explicit way to recover into manual selection when an
interactive boot was configured to try policy first.

## What Changes

- Add an operator-facing policy signing helper or documented example flow that
  preserves the Ed25519 signed-policy contract without requiring ad hoc local
  scripts.
- Add remote signed policy URL support with authenticated cache fallback.
- Add an end-to-end signed policy smoke that selects a real catalog target,
  validates options, and exercises redacted diagnostics.
- Add an explicit interactive fallback option that returns menu-mode runs to
  manual target selection when policy evaluation fails.

## Capabilities

### Modified Capabilities

- `bootup-dynamic-policy-resolution`: extend policy sources from signed local
  files to signed HTTPS URLs with cache fallback, define policy signing
  ergonomics, add smoke coverage, and specify explicit interactive fallback.
- `bootup-persistent-diagnostics`: include remote policy source posture and
  fallback category while preserving existing redaction rules.

## Impact

- Affected code: CLI flags, policy source loading, policy cache handling,
  policy signing tooling, diagnostics summaries, docs, and focused tests.
- Operator behavior: remote policy still requires local trust material; cache
  entries are only usable after the same authentication and freshness checks as
  fresh responses.
- Dependencies: no new third-party dependency is expected.
