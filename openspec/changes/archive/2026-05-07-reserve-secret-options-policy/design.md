## Context

Bootup target options are intentionally simple data. They expand into Linux
kernel command-line fragments or FreeBSD `loader.kboot` arguments, and those
values are visible in `plan-target`, `stage-target`, and smoke logs. This is
acceptable for non-secret installer settings such as serial console, mirror
URL, text mode, and mfsBSD hostname, but it is not safe for root passwords,
password hashes, SSH keys, tokens, or policy-generated secrets.

The docs also describe a future fully dynamic policy mode. That mode would need
clear trust, redaction, and failure semantics before it can decide boot actions
or inject sensitive values in stage-1.

## Goals / Non-Goals

**Goals:**

- Reserve an explicit `secret` marker on target option definitions and reject
  it with a clear validation error.
- Make current target option semantics explicit: non-secret command-line or
  loader-argument data only.
- Keep dynamic policy execution out of static catalogs and provider runtime
  config.
- Document the future shape of dynamic policy without implementing it here.

**Non-Goals:**

- Add password, password-hash, SSH-key, token, or file-secret injection.
- Add policy server calls, local policy scripts, remote plugins, or embedded
  policy interpreters.
- Redesign `BootPlan` around redacted display fields in this change.

## Decisions

- Add a `Secret bool` field to `provider.TargetOption` with JSON name
  `secret`. Validation rejects `secret: true` before command-line behavior is
  considered. This gives catalog authors an explicit error instead of accepting
  a field whose semantics bootup cannot honor safely.
- Keep the field out of generated default catalogs by relying on the existing
  zero-value JSON omission. Non-secret options continue to behave exactly as
  they do today.
- Treat dynamic policy as a future executor/resolver capability, not as a
  catalog or provider-config extension. Static catalogs remain data-only and
  provider runtime config remains typed operator input for compiled-in provider
  behavior.
- Document the future policy direction in a dedicated operator/security note so
  password and SSH-key requests have a clear next-design target.

## Risks / Trade-offs

- Reserving `secret` now does not deliver secrets yet -> the explicit rejection
  keeps users from thinking command-line delivery is safe.
- A future implementation may choose a different secret delivery shape -> the
  marker can still remain the opt-in signal for redacted option metadata.
- Catalog authors may need to remove `secret: true` from experiments -> the
  error explains why and points them toward a future dedicated capability.
