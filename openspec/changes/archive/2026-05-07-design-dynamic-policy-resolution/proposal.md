## Why

Bootup has static catalogs and provider discovery, but no safe dynamic policy
mode for automated target selection. Adding policy without a narrow contract
would risk remote code execution, silent target drift, secret leakage, and
unclear behavior when a policy source is unavailable.

## What Changes

- Design a dynamic policy resolver that consumes authenticated policy
  decisions as data, not code.
- Require explicit local trust material, bounded evaluation, and fail-closed
  behavior.
- Limit policy output to an existing target ID, selected non-secret options,
  and optional secret references handled by the separate secret input design.
- Define diagnostics, cache/freshness posture, and unsupported-output behavior
  before implementation starts.

## Capabilities

### New Capabilities

- `bootup-dynamic-policy-resolution`: trusted, data-only runtime policy
  decisions for selecting already-known boot targets.

### Modified Capabilities

- `bootup-provider-runtime-config`: provider runtime configuration remains
  provider data; dynamic policy is configured and evaluated through a separate
  resolver capability.

## Impact

- Affected future code: CLI/config loading, policy document verification,
  target inventory rendering, target selection, diagnostics, and non-interactive
  execution.
- Public behavior when implemented: policy failure fails closed unless the
  operator is in an interactive mode that can return to manual selection.
- Dependencies: no new dependency is required by the design.
