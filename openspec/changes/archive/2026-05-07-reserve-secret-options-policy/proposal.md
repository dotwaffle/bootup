## Why

Target options are currently expanded into Linux command lines or FreeBSD
loader arguments that are printed in plan and stage diagnostics. Before adding
passwords, SSH keys, or dynamic policy output, bootup needs an explicit safety
boundary that fails closed instead of accidentally accepting secret-bearing
catalog data.

## What Changes

- Reserve a `secret` marker on target option definitions and reject any catalog
  option that sets it until bootup has a secret-safe delivery channel.
- Document that current target options are non-secret command-line or loader
  argument data.
- Document the dynamic policy boundary: no runtime scripts, remote plugins, or
  policy service decisions in static catalogs or provider runtime config.
- Capture the future policy direction so the next implementation can be
  designed around redacted diagnostics and explicit trust material.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `bootup-target-options`: target options remain non-secret; secret-bearing
  option definitions are rejected.
- `bootup-provider-runtime-config`: runtime configuration remains declarative
  provider data, not executable dynamic policy.

## Impact

- Affected code: `internal/provider`, catalog tests, provider tests, and
  operator documentation.
- Public behavior: catalogs that set `secret: true` on a target option fail
  validation with an explicit error.
- Dependencies: no new third-party dependencies.
