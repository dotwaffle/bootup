# Provider catalog model

Bootup providers are compiled into the stage-1 image. The current catalog model
implements static, concrete boot targets: the target list is known by the
bootup binary or its bundled static catalog content, and target IDs stay stable
until the tool or catalog content is updated.

Each static target carries typed catalog metadata:

- distribution, for example `debian` or `ubuntu`
- release, for example `trixie` or `26.04`
- architecture, currently `amd64`
- kind, for example `installer`

The operator interfaces use that metadata for grouping and labels. Providers
still own boot planning and artifact staging, so the catalog describes what can
be selected while provider code decides how to resolve, verify, and stage it.

## Implemented mode: static concrete targets

This mode is intentionally simple. Choosing a target such as Debian trixie amd64
netboot always selects that concrete target. New releases or architectures do
not appear until bootup itself or future static catalog content is updated.

Bootup embeds a default static catalog. The current default catalog includes:

- `debian-bookworm-amd64-netboot`
- `debian-trixie-amd64-netboot`
- `ubuntu-2604-amd64-netboot`

Operators can replace the embedded catalog with a local JSON file using
`--catalog`. Replacement is all-or-nothing: a supplied catalog becomes the
complete static target list for compiled-in providers. Bootup validates the
catalog before provider registration and rejects malformed JSON, unsupported
schema versions, duplicate target IDs, incomplete target metadata, and provider
IDs that are not compiled into the binary.

Catalog documents use schema version 1:

```json
{
  "schema_version": 1,
  "targets": [
    {
      "id": "debian-trixie-amd64-netboot",
      "provider_id": "debian",
      "name": "Debian trixie amd64 netboot",
      "catalog": {
        "distribution": "debian",
        "release": "trixie",
        "architecture": "amd64",
        "kind": "installer"
      }
    }
  ]
}
```

The document is data only. It selects concrete targets for provider code that is
already compiled into bootup; it cannot load provider plugins or executable
policy.

## Future mode: hosted static catalogs

A hosted catalog can use the same static target model, but bootup does not fetch
catalogs from URLs yet. URL loading needs a separate authenticity and freshness
design covering signatures or pins, cache behavior, offline fallback, and
operator trust configuration.

Until then, operators that want hosted content should fetch or generate the
catalog outside bootup and pass the resulting local file with `--catalog`.

## Future mode: dynamic distro discovery

A later provider mode can expose a distro family first, then discover available
releases, architectures, variants, and install options when the operator selects
that provider. That mode can also decorate results with external lifecycle data
such as end-of-life status.

That discovery logic is deliberately outside the current static catalog
contract. It should have a separate provider discovery contract so static
catalog documents remain stable concrete target lists.

## Future mode: dynamic policy

A fully dynamic mode can evaluate site-specific policy before choosing a boot
action. That policy might call an in-house service, use machine identity such as
MAC address or serial number, decide to boot local disk, or choose an installer
with generated options.

Bootup does not implement script execution, remote policy plugins, or a
self-hosted catalog/policy server yet. Those pieces should be designed as
separate capabilities so the static catalog remains predictable and usable in
restricted stage-1 environments.
