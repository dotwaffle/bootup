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

Bootup supports an initial signed local dynamic policy mode. A policy decision
is JSON data, authenticated with a detached Ed25519 signature over the raw
policy bytes:

```json
{
  "schema_version": 1,
  "decision_id": "site-a-rack-22-node-03",
  "target_id": "ubuntu-2604-amd64-netboot",
  "options": {
    "console": "serial"
  },
  "secret_refs": {
    "installer-password": "site-installer-password"
  },
  "published_at": "2026-05-07T10:00:00Z",
  "expires_at": "2026-05-07T10:10:00Z"
}
```

Use `policy-target` to resolve and print the selected boot plan:

```sh
bootup --mode=policy-target \
  --policy-file=/etc/bootup/policy.json \
  --policy-signature=/etc/bootup/policy.json.sig \
  --policy-public-key=/etc/bootup/policy.pub
```

The policy flags can also be used with `plan-target`, `stage-target`, or
`boot-target`; in those modes the policy supplies the target, selected
non-secret options, and secret references. Do not combine dynamic policy with
`--target`, `--option`, or `--discovery-family`.

Policy decisions fail closed before provider planning when trust material is
missing, signature verification fails, JSON is malformed, freshness metadata is
missing, the decision is expired, `--policy-max-age` is exceeded, or the result
references an unknown target, invalid option, undeclared secret, or missing
required secret. `expires_at` is accepted as a freshness bound; `published_at`
is required when `--policy-max-age` is used. `--policy-cache` updates a local
cache after a fresh source decision authenticates, and
`--policy-cache-fallback` can use that cache after source read failure. Cached
bytes go through the same signature and freshness checks.

Policy is data only. It cannot define targets, providers, boot actions,
command-line fragments, trust roots, artifact hash pins, plugins, scripts,
Rego, WebAssembly, shell commands, or remote provider code. Remote policy URLs
and interactive manual fallback remain deferred.
