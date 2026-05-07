## Why

Bootup currently rejects secret target options because option values are
expanded into boot arguments and diagnostics. Operators still need a supported
path for installer passwords, SSH keys, tokens, and policy-selected secret
references that does not print or persist secret material.

## What Changes

- Design an explicit secret input capability that keeps target options
  non-secret.
- Introduce target/provider secret declarations keyed by stable secret IDs.
- Require operators to provide secret values through local file-backed inputs,
  not inline catalog, command-line, or environment values.
- Define redaction, diagnostics, staging, validation, and failure behavior for
  secret inputs before implementation work starts.

## Capabilities

### New Capabilities

- `bootup-secret-input-delivery`: secret-safe input collection and delivery for
  compiled-in providers.

### Modified Capabilities

- `bootup-target-options`: current target options remain non-secret; secret
  delivery is a separate capability instead of an option fragment extension.

## Impact

- Affected future code: CLI config, provider planning interfaces, diagnostics,
  staging, catalog validation, and provider-specific secret consumers.
- Public behavior when implemented: required missing secrets fail before
  planning or staging; secret values never become boot arguments or diagnostic
  content.
- Dependencies: no new dependency is required by the design.
