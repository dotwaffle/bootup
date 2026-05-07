## Context

Bootup already renders status, plan, stage, and fatal output to the active
console and keeps the environment available after failures. That is useful
during an interactive session, but failures from QEMU smokes, serial consoles,
or field boots can still be hard to reconstruct after the console scrollback is
gone. The long-term logging question should start with a local, explicit,
operator-controlled diagnostic bundle rather than remote logging.

## Goals / Non-Goals

**Goals:**

- Add an opt-in diagnostics directory that records failure context without
  changing normal console output.
- Capture enough context to reproduce or triage the failure: mode, target,
  discovery family, selected option IDs, catalog source posture, provider
  config path presence, stdout, stderr/log output, and final error.
- Keep diagnostics best-effort and never mask the original boot failure.
- Avoid persisting selected option values or provider config contents in this
  first slice.

**Non-Goals:**

- Add remote log shipping, persistent disks, journal integration, or metrics.
- Redesign plan/stage output redaction.
- Capture provider config bytes, trust material bytes, selected option values,
  SSH keys, passwords, or tokens.

## Decisions

- Implement diagnostics at the CLI boundary. `runWithIO` already receives the
  real stdout/stderr writers and knows the parsed startup flags. Wrapping those
  writers lets bootup preserve normal console output while buffering the same
  data for failure reports.
- Write a JSON report plus `stdout.txt` and `stderr.txt`. JSON gives automated
  tooling a stable summary; text files preserve operator-facing streams without
  forcing every message into structured fields.
- Use a timestamped report directory under the configured diagnostics directory.
  This avoids overwriting earlier failures and keeps the output easy to copy
  from a VM or rescue shell.
- Treat diagnostics write failures as secondary errors. The returned error
  should still be the original boot failure, with diagnostics write failure
  context appended only after the original error is preserved.

## Risks / Trade-offs

- Buffered stdout/stderr can grow in long sessions -> diagnostics only buffers
  when explicitly enabled and is intended for failure triage.
- Existing outputs can contain non-secret option values -> the report captures
  current diagnostics by design, while metadata avoids option values and config
  contents.
- A failure could happen before flags parse -> diagnostics cannot activate
  until `--diagnostics-dir` is parsed.
