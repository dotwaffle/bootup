## Context

The first dynamic policy implementation intentionally limited scope to signed
local policy files. That proved the data-only decision model, but leaves three
operational gaps: producing signed policy files is manual, hosted policy cannot
be consumed directly, and policy failure in an interactive boot still behaves
like a hard startup failure.

This change keeps the same security boundary. Policy remains signed data that
selects existing targets and validated options. It does not introduce remote
code, provider definitions, arbitrary boot arguments, or inline secrets.

## Goals / Non-Goals

Goals:

- Provide a repeatable local signing flow for Ed25519 policy decisions.
- Fetch signed policy decisions from HTTPS URLs with local trust material.
- Cache only authenticated policy bytes and use cache fallback only after
  revalidating signature and freshness.
- Exercise the signed policy path with a smoke that uses a real catalog target
  and diagnostics output.
- Allow menu-mode operators to opt into manual fallback when policy fails.

Non-goals:

- Support executable policy engines, plugins, Rego, shell, Lua, WebAssembly, or
  remote provider code.
- Let remote policy define new targets, change artifact trust, inject kernel
  command-line fragments, or override provider validation.
- Treat HTTPS transport as a replacement for signed policy trust material.
- Use unauthenticated or expired cache content as a fallback.

## Policy Source Shape

Bootup will support exactly one policy source per run:

- `--policy-file`: existing signed local decision bytes.
- `--policy-url`: signed remote decision bytes fetched over HTTPS.

Both source kinds use the existing detached signature and public key contract.
The remote URL source may configure a request timeout and an optional cache
file. Fetch failures, non-2xx responses, malformed responses, signature
failures, expired decisions, and maximum-age failures all fail closed unless an
explicit interactive fallback is selected.

Remote cache fallback stores the policy response body after a fresh response
has passed authentication and freshness checks. When a later fetch fails, the
cache body may be loaded, but it must pass the same signature and freshness
checks before it can influence target planning.

## Signing Helper

The helper should use Go's standard Ed25519 implementation and write raw key and
signature files compatible with the bootup runtime flags:

- Generate a private/public key pair with local file permissions appropriate
  for private trust material.
- Sign a policy JSON document into a detached signature file.
- Avoid embedding private key bytes in logs, diagnostics, or docs.

Documentation should show the helper flow and the equivalent runtime flags for
local and remote policy runs.

## Interactive Fallback

Policy remains fail-closed by default. Interactive fallback must be explicit,
for example through a `none`/`manual` style flag. The fallback is only valid for
interactive menu startup. When selected and policy evaluation fails, bootup
reports a concise policy failure category and starts the normal manual target
menu instead of planning a target from policy data.

Policy success in a policy-first menu run may proceed directly to the selected
target after normal target and option validation. Policy failure in
non-interactive modes remains a hard error.

## Diagnostics

Diagnostics may include source kind, redacted source location posture, policy
decision ID, target ID, selected option IDs, freshness timestamps, cache usage
category, and fallback category. Diagnostics must not include remote response
bodies, selected option values, secret values, secret paths, private keys,
public key bytes, detached signature bytes, or bearer-style credentials
embedded in URLs.
