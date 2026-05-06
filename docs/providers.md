# Provider catalog model

Bootup providers are compiled into the stage-1 image. The catalog model
implements static, concrete boot targets: the target list is known by the
bootup binary or its bundled static catalog content, and target IDs stay stable
until the tool or catalog content is updated. Providers can also opt into
runtime discovery of concrete targets through compiled-in Go code.

Each static target carries typed catalog metadata:

- distribution, for example `debian`, `fedora`, `opensuse`, or `gparted`
- release, for example `trixie` or `26.04`
- architecture, currently `amd64`
- kind, for example `installer`, `tool`, or `localboot`
- optional boot action, such as `localboot`; omitted means `linux-kexec`
- optional source facts such as a target source URL, installer ISO filename,
  kernel path, initrd path, or command line
- optional target option definitions that validate operator-selected values
  and append command-line fragments

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
- `debian-forky-amd64-netboot`
- `fedora-43-amd64-server-netboot`
- `fedora-44-amd64-server-netboot`
- `local-disk-auto`
- `opensuse-leap-160-amd64-netboot`
- `archlinux-latest-amd64-netboot`
- `gparted-live-1813-amd64`
- `ubuntu-24044-amd64-netboot`
- `ubuntu-2510-amd64-netboot`
- `ubuntu-2604-amd64-netboot`

The embedded `internal/catalog/default.json` file is generated from
`internal/catalog/source.json`. Update the source file and run:

```sh
go generate ./internal/catalog
```

Repository tests fail when the generated file is stale.

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
      "id": "opensuse-leap-160-amd64-netboot",
      "provider_id": "linux",
      "name": "openSUSE Leap 16.0 amd64 installer",
      "catalog": {
        "distribution": "opensuse",
        "release": "leap-16.0",
        "architecture": "amd64",
        "kind": "installer"
      },
      "source": {
        "base_url": "https://download.opensuse.org/distribution/leap/16.0/repo/oss",
        "kernel_path": "boot/x86_64/loader/linux",
        "initrd_path": "boot/x86_64/loader/initrd",
        "cmdline": "netsetup=dhcp install={base_url} console=ttyS0"
      },
      "options": [
        {
          "id": "text-install",
          "label": "Text install",
          "type": "bool",
          "fragment": "textmode=1"
        },
        {
          "id": "mirror-url",
          "label": "Installer mirror URL",
          "type": "string",
          "template": "install={value}"
        }
      ]
    },
    {
      "id": "local-disk-auto",
      "provider_id": "local",
      "name": "Boot from local disk",
      "action": "localboot",
      "catalog": {
        "distribution": "local",
        "release": "disk",
        "architecture": "amd64",
        "kind": "localboot"
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
providers that need one. The generic `linux` provider also accepts
`source.kernel_path`, optional `source.initrd_path`, and `source.cmdline`.
Those paths are clean relative URL paths resolved against `source.base_url`.
The command line may include `{base_url}` to refer to the trimmed source root.
`lifecycle` is informational decoration for operator display; it is not
signature, checksum, transport, keyring, or other trust material.

Target options are non-executable catalog data. Supported option types are
`bool`, `enum`, and `string`. Boolean options append their `fragment` only when
selected as `true`. Enum options define allowed `values`, each with an optional
fragment. String options expand exactly one `{value}` placeholder in
`template`. Bootup rejects duplicate option IDs, unsupported types, invalid enum
values, and fragments or templates with surrounding whitespace or control
characters.

Operators select options with repeatable `--option id=value` flags. Selected
fragments are appended after provider defaults and before `--append-cmdline`,
so operator append text remains the final command-line addition.

The `local` provider exposes `local-disk-auto` as a `localboot` action. It does
not download artifacts; handoff invokes u-root's local boot command to inspect
local boot configuration and continue from disk.

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
bootup --mode=discover-targets --discovery-family=ubuntu
```

The Debian provider discovers amd64 netboot installers from the configured
mirror or `discovery_url`. It fetches the `dists/` index, filters release
aliases such as `stable` and `testing`, probes each release for amd64 netboot
checksum metadata, and returns one target per available release.

The Ubuntu provider discovers amd64 netboot installers from the configured
release index. It reads release links, checks each release `SHA256SUMS` file
for a live-server amd64 ISO, probes the `netboot/amd64/linux` and
`netboot/amd64/initrd` paths, and returns concrete targets that the normal
Ubuntu planner can stage.

The Fedora provider currently uses static catalog targets. For each Fedora
Server target, planning resolves `images/pxeboot/vmlinuz`,
`images/pxeboot/initrd.img`, and an `inst.repo=` command line from the target
source URL or an operator-supplied `release_url` override.

The generic Linux provider currently handles openSUSE Leap, Arch Linux, and
GParted Live catalog targets. These are Linux-shaped paths: kernel plus
optional initrd plus command line, staged over HTTPS.

MemTest86+ 8.00 is intentionally not in the default catalog. Its x86_64 image
boots when a firmware or bootloader enters the Linux boot protocol directly,
but it does not satisfy the currently implemented Linux kexec handoff paths.
It should be reintroduced only with a dedicated bootloader-style handoff, a
known kexec-compatible image, or a proven real-mode `kexec_load` loader.

Discovery is timeout-bound and explicit. Providers accept optional
`discovery_url`, `discovery_timeout`, and lifecycle decoration in
`--provider-config`. If discovery fails, the already-loaded static catalog
targets remain available.

Discovery results can carry lifecycle decoration such as `supported`,
`obsolete`, `eol`, or `unknown`. That metadata is displayed to operators as
information only. Providers must not treat lifecycle metadata as signature,
checksum, keyring, transport, or other trust material.

## Future mode: hosted static catalogs

A hosted catalog can use the same static target model, but bootup does not fetch
catalogs from URLs yet. URL loading needs a separate authenticity and freshness
design covering signatures or pinned digests, cache behavior, offline fallback,
and operator trust configuration. Catalog authenticity is separate from
distribution artifact verification; operators still configure provider trust
material for downloaded boot artifacts.

Until then, operators that want hosted content should fetch or generate the
catalog outside bootup and pass the resulting local file with `--catalog`.

## Future mode: broader distro discovery

Provider discovery can expand beyond the current Debian and Ubuntu amd64 netboot
targets to additional distributions, architectures, variants, install options,
and optional lifecycle data sources. Fedora is static-only for now. That logic
remains outside the static catalog contract so static catalog documents stay
stable concrete target lists.

BSD installers, HDT, memdisk ISO images, syslinux COM32 modules, and iPXE
chainload flows are intentionally deferred. The salstar BSD and several tool
paths depend on bootloader semantics that are not the same as Linux
kernel/initrd kexec. u-root's Multiboot support helps only for payloads that
are actually Multiboot-compatible; stock FreeBSD 15.0 release artifacts still
need a FreeBSD loader, EFI chainload, disk/ISO chainload, or another dedicated
handoff. These targets should be added only after bootup has a dedicated
executor family for those handoff types.

## Future mode: dynamic policy

A fully dynamic mode can evaluate site-specific policy before choosing a boot
action. That policy might call an in-house service, use machine identity such as
MAC address or serial number, decide to boot local disk, or choose an installer
with generated options.

Bootup does not implement script execution, remote policy plugins, or a
self-hosted catalog/policy server yet. Those pieces should be designed as
separate capabilities so the static catalog remains predictable and usable in
restricted stage-1 environments.
