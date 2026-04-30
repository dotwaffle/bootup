# Provider catalog model

Bootup providers are compiled into the stage-1 image. The catalog model
implements static, concrete boot targets: the target list is known by the
bootup binary or its bundled static catalog content, and target IDs stay stable
until the tool or catalog content is updated. Providers can also opt into
runtime discovery of concrete targets through compiled-in Go code.

Each static target carries typed catalog metadata:

- distribution, for example `debian` or `ubuntu`
- release, for example `trixie` or `26.04`
- architecture, currently `amd64`
- kind, for example `installer`
- optional source facts such as a target source URL or installer ISO filename

The operator interfaces use that metadata for grouping and labels. Providers
still own boot planning and artifact staging, so the catalog describes what can
be selected while provider code decides how to resolve, verify, and stage it.

## Implemented mode: static concrete targets

This mode is intentionally simple. Choosing a target such as Debian trixie amd64
netboot always selects that concrete target. New releases or architectures do
not appear until bootup itself or future static catalog content is updated.

Bootup embeds a default static catalog. The current default catalog includes:

- `debian-bullseye-amd64-netboot`
- `debian-bookworm-amd64-netboot`
- `debian-trixie-amd64-netboot`
- `ubuntu-24044-amd64-netboot`
- `ubuntu-2510-amd64-netboot`
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
    },
    {
      "id": "ubuntu-24044-amd64-netboot",
      "provider_id": "ubuntu",
      "name": "Ubuntu 24.04.4 amd64 netboot",
      "catalog": {
        "distribution": "ubuntu",
        "release": "24.04.4",
        "architecture": "amd64",
        "kind": "installer"
      },
      "source": {
        "base_url": "https://releases.ubuntu.com/24.04",
        "iso_name": "ubuntu-24.04.4-live-server-amd64.iso"
      }
    }
  ]
}
```

The document is data only. It selects concrete targets for provider code that is
already compiled into bootup; it cannot load provider plugins or executable
policy. `source.base_url` is an absolute HTTP(S) provider source root for that
target, and `source.iso_name` is a pathless installer ISO filename used by
providers that need one.

## Implemented mode: provider discovery

Dynamic distro discovery is additive to the static catalog. Discovery-capable
providers expose a family entry such as `debian`; selecting that family runs
compiled-in provider logic and returns normal concrete `provider.Target` values.
Planning, verification, staging, and kexec handoff still use the same provider
path as static targets.

Menu mode renders discovery families alongside static concrete targets. In the
plain and rich UIs, selecting a family runs discovery and opens a second target
selection menu. Non-interactive diagnostics can list discovered targets without
staging artifacts:

```sh
bootup --mode=discover-targets --discovery-family=debian
```

The Debian provider currently discovers amd64 netboot installers from the
configured mirror. It fetches the mirror `dists/` index, filters release aliases
such as `stable` and `testing`, probes each release for amd64 netboot checksum
metadata, and returns one target per available release. Discovery is
timeout-bound and uses the provider's configured mirror URL. If discovery
fails, the already-loaded static catalog targets remain available.

Discovery results can carry lifecycle decoration such as `supported`,
`obsolete`, `eol`, or `unknown`. That metadata is displayed to operators as
information only. Providers must not treat lifecycle metadata as signature,
checksum, keyring, transport, or other trust material.

## Future mode: hosted static catalogs

A hosted catalog can use the same static target model, but bootup does not fetch
catalogs from URLs yet. URL loading needs a separate authenticity and freshness
design covering signatures or pins, cache behavior, offline fallback, and
operator trust configuration.

Until then, operators that want hosted content should fetch or generate the
catalog outside bootup and pass the resulting local file with `--catalog`.

## Future mode: broader distro discovery

Future provider discovery can expand beyond Debian amd64 netboot targets to
additional distributions, architectures, variants, install options, and optional
lifecycle data sources. That logic remains outside the static catalog contract
so static catalog documents stay stable concrete target lists.

## Future mode: dynamic policy

A fully dynamic mode can evaluate site-specific policy before choosing a boot
action. That policy might call an in-house service, use machine identity such as
MAC address or serial number, decide to boot local disk, or choose an installer
with generated options.

Bootup does not implement script execution, remote policy plugins, or a
self-hosted catalog/policy server yet. Those pieces should be designed as
separate capabilities so the static catalog remains predictable and usable in
restricted stage-1 environments.
