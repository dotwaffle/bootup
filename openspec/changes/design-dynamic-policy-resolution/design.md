## Context

Dynamic policy could automate boot target selection based on site, machine, or
operator criteria. The unsafe version would be a remote script, plugin, or
service response that can run code or inject arbitrary boot arguments. Bootup's
current model deliberately avoids that: providers are compiled in, catalogs are
data, and target options are validated before planning.

The policy design must preserve those boundaries and compose with the proposed
secret input delivery capability.

## Goals / Non-Goals

Goals:

- Resolve a boot decision from trusted data under a configured timeout.
- Fail closed when policy is unavailable, unauthenticated, expired, malformed,
  or references unsupported targets/options.
- Keep policy output data-only: target ID, non-secret option selections, and
  secret input references.
- Require explicit operator trust material for every remote or mutable policy
  source.
- Keep policy evaluation observable without printing secret values or
  unbounded service responses.

Non-goals:

- Execute scripts, WebAssembly, shell snippets, Lua, Rego, plugins, or remote
  provider code in stage-1.
- Let policy documents define new targets, providers, command-line fragments,
  boot actions, trust roots, or artifact verification overrides.
- Let policy output carry secret values inline.
- Replace interactive menu mode; policy is an additional startup mode.

## Decision Shape

The policy resolver should operate after static catalogs and selected discovery
families have produced a target inventory. The result is a small typed
decision:

```json
{
  "schema_version": 1,
  "decision_id": "site-a-rack-22-node-03",
  "target_id": "ubuntu-2604-amd64-netboot",
  "options": {
    "console": "serial",
    "mirror-url": "https://mirror.example/ubuntu"
  },
  "secret_refs": {
    "installer-password": "site-installer-password"
  },
  "published_at": "2026-05-07T10:00:00Z",
  "expires_at": "2026-05-07T10:10:00Z"
}
```

`options` are ordinary non-secret target option selections and must validate
against the selected target. `secret_refs` map target-declared secret IDs to
operator-provided secret input IDs; they do not contain values.

## Trust And Freshness

- Local or remote policy decisions must be authenticated before use. The
  recommended first implementation is a signed envelope using a configured
  Ed25519 public key, reusing the hosted catalog signature posture where
  practical.
- HTTPS transport is required for remote policy sources, but transport security
  is not sufficient by itself.
- Decisions include freshness metadata. Expired decisions fail closed. Operators
  may configure a maximum age to limit stale decisions.
- Cache fallback is optional and must reuse the same authentication and
  freshness checks as a freshly fetched decision.

## Failure Behavior

- Non-interactive policy mode fails before planning when policy cannot produce
  a valid decision.
- Interactive menu mode may report the policy failure and return to manual
  target selection if the operator explicitly selected that fallback behavior.
- Decisions referencing unknown targets, unsupported options, invalid option
  values, undeclared secret IDs, missing required secrets, or unsupported boot
  actions fail before staging.
- Timeout, context cancellation, malformed JSON, failed signatures, missing
  trust material, and expired decisions fail closed.

## Diagnostics

Diagnostics may include policy source posture, decision ID, target ID, selected
option IDs, secret reference IDs, freshness timestamps, and failure categories.
Diagnostics must not include policy response bodies, secret values, secret input
paths, provider config contents, trust private material, or unredacted option
values marked non-displayable by a future provider contract.

## Open Questions

- Whether the first implementation should support both `policy_file` and
  `policy_url`, or start with signed local files only.
- Whether policy should run only over static catalog targets initially, or allow
  an explicit discovery-before-policy step.
- Whether the policy trust envelope should be shared with hosted catalogs or
  live in a separate package to keep cache/freshness behavior independent.
