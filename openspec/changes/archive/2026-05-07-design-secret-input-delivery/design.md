## Context

Target options are visible by design. They may become Linux command-line
fragments, FreeBSD loader arguments, dry-run output, smoke logs, or diagnostics.
That makes them useful for installer knobs but unsafe for passwords, password
hashes, SSH keys, API tokens, and policy-generated secret references.

Bootup needs a dedicated secret path before dynamic policy or installer
automation can carry sensitive material.

## Goals / Non-Goals

Goals:

- Keep secret bytes out of catalogs, provider runtime config, CLI arguments,
  environment variables, kernel command lines, loader arguments, logs, smoke
  output, and diagnostics.
- Let targets and providers declare required or optional secret IDs with
  operator-facing labels and purposes.
- Let operators provide secret values as local filesystem paths in the stage-1
  environment.
- Fail closed when a required secret is missing, unreadable, oversized, or
  explicitly unsafe.
- Provide providers with secret handles or staged private files, not raw values
  in public boot plan fields.

Non-goals:

- Add an interactive password prompt in the first implementation.
- Add secret managers, network secret fetches, TPM unsealing, or agent plugins.
- Support secret interpolation into target option fragments.
- Support arbitrary provider scripts that consume secrets.

## Decisions

- Add a new secret declaration shape separate from target options. A
  declaration has an ID, label, purpose, required flag, and provider-owned
  delivery hint. The ID is stable catalog/provider data; the value is not.
- Add an operator input map such as `--secret id=/absolute/path`, with the
  exact CLI spelling left to implementation. Values are filesystem paths only.
  Inline values and environment variable expansion are out of scope because
  they leak through process listings, shell history, or diagnostics.
- Validate secret input paths before planning. Paths must be absolute, local,
  regular files, below a conservative size limit, and readable by bootup. The
  implementation should reject group/other-readable files by default and offer
  an explicit operator override only if ISO/FAT-style media makes POSIX mode
  checks unreliable.
- Pass secrets to providers through an explicit secret store associated with
  planning/staging. Public `BootPlan` fields may include secret IDs and delivery
  status, but not bytes, source paths, or derived command-line values.
- Providers that need file delivery receive staged private files under the
  target staging directory with mode `0600`. Those staged paths are treated as
  sensitive and redacted from diagnostics unless a provider-specific consumer
  can prove they are not exposed after handoff.
- Diagnostics may include declared secret IDs, whether each required secret was
  supplied, and validation failure categories. They must not include secret
  values, input paths, staged paths, hashes of secret values, or provider config
  contents.
- Existing `secret: true` target options remain rejected. A future catalog can
  use secret declarations plus non-secret options together, but secrets never
  expand directly into option templates.

## Failure Behavior

- Missing required secret: fail before provider planning.
- Missing optional secret: continue only if the provider explicitly marks it
  optional.
- Invalid, oversized, or unsafe secret file: fail before provider planning.
- Provider asks for a secret ID that the target did not declare: fail before
  staging.
- Diagnostics write failure: preserve the original boot failure and report the
  diagnostics failure as secondary context, without dumping secret metadata.

## Open Questions

- Whether the first implementation should expose secret inputs only through
  CLI flags or also through a local runtime config document.
- Whether POSIX permission rejection should be hard-fail on all filesystems or
  allow a narrowly named override for read-only boot media.
- Which first provider should consume the capability: mfsBSD root password
  replacement, SSH authorized keys, or an automated Linux installer secret.
