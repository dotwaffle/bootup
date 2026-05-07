# Policy and Secret Inputs

Bootup's current provider and catalog inputs are declarative data. Static
catalogs, hosted catalogs, target options, and provider runtime configuration
do not execute policy scripts, load runtime plugins, or call a policy service to
choose a boot target.

## Target Options

Target options are non-secret boot argument data. Selected values can become
Linux kernel command-line fragments or FreeBSD `loader.kboot` arguments, and
bootup prints those resulting values in plan, stage, smoke, and diagnostic
output.

Use target options for values that are safe to inspect on an operator console,
such as serial console choice, text install mode, installer mirror URL, or
rescue hostname. Do not use target options for passwords, password hashes, SSH
keys, API tokens, or policy-generated secrets.

The `secret` target option marker remains rejected. Catalogs that set
`"secret": true` fail validation because secret material must use the separate
secret input path.

## Secret Inputs

Secret inputs are provider-owned declarations separate from target options. A
target can declare an ID, label, purpose, whether the input is required, and the
delivery mode. The current delivery mode is `staged-file`.

Operators provide values with repeatable local file-backed flags:

```sh
bootup --mode=stage-target --target=site-installer --secret installer-password=/run/bootup/secrets/installer-password
```

Inline values, environment expansion, provider runtime config values, and
target option fragments are not supported for secrets. Bootup validates
absolute local regular files before provider planning, rejects group- or
other-readable files by default, and fails closed when a required secret is
missing or unsafe.

Diagnostics may include the secret ID and reference ID. They do not include the
secret value, the input path, staged file path, value hash, provider config
contents, or derived boot arguments containing secret material. The default
catalog does not currently include a distro provider target that consumes a
secret input.

## Dynamic Policy

Dynamic policy is still a future capability. A safe policy design needs all of
these pieces before it should affect boot decisions:

- explicit local trust material for any policy service or policy document
- timeout-bound and fail-closed evaluation
- a data-only result shape, such as target ID plus selected non-secret options
- a separate delivery path for secret material
- redacted plan, stage, smoke, and error output
- clear behavior when policy is unavailable or returns an unsupported target

Until that exists, bootup rejects unsupported policy fields in provider runtime
configuration and keeps static catalogs predictable.
