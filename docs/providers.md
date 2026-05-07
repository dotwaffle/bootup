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
- optional secret input declarations that describe provider-owned sensitive
  inputs without embedding secret values

The operator interfaces use that metadata for grouping and labels. Providers
still own boot planning and artifact staging, so the catalog describes what can
be selected while provider code decides how to resolve, verify, and stage it.

## Implemented mode: static concrete targets

This mode is intentionally simple. Choosing a target such as Debian trixie amd64
netboot always selects that concrete target. In static mode, new releases or
architectures do not appear until the embedded default catalog or a selected
static catalog is updated. Provider discovery families are the separate runtime
path for finding supported releases.

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
- `mfsbsd-142-amd64`
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
`--catalog`, or with an authenticated hosted JSON file using `--catalog-url`.
Replacement is the default: a supplied local or hosted catalog becomes the
complete static target list for compiled-in providers. Add
`--catalog-include-default` to compose the embedded default catalog with the
selected local or hosted catalog. Composition is additive only, and duplicate
target IDs across sources are rejected. Bootup validates the catalog before
provider registration and rejects malformed JSON, unsupported schema versions,
duplicate target IDs, incomplete target metadata, and provider IDs that are not
compiled into the binary.

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
        "kernel_sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
        "initrd_sha256": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
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

Run the catalog matrix to audit the configured catalog after local or hosted
catalog selection and provider registration:

```sh
bootup --mode=catalog-matrix
```

The matrix is tab-separated and includes target ID, provider, distribution,
release, architecture, kind, lifecycle status, resolved boot action, dry-run
plan status, artifact trust posture, smoke coverage, and any planning error.
It requests offline provider planning only; it does not download artifacts,
stage files, contact upstream mirrors, fetch remote metadata, or launch QEMU.
Planning errors are rendered in the matrix and make the command exit nonzero.

Artifact trust labels describe the dry-run boot plan:

- `hash-pinned`: every downloadable artifact has a SHA-256 pin.
- `signed-metadata`: artifact verification uses signed metadata.
- `release-metadata`: artifact verification uses release metadata and
  checksums.
- `https-only`: downloads use HTTPS without stronger planned artifact trust.
- `partial-hashes`: only some planned artifacts have SHA-256 pins.
- `not-applicable`: the plan does not download boot artifacts.
- `unverified`: planned downloads are not hash-pinned, metadata-backed, or
  HTTPS-only.

Smoke coverage labels identify explicit helper support:

- `live-stage`: `BOOTUP_LIVE_CATALOG_SMOKE=1 go test ./test/live` can stage the
  target outside a VM.
- `catalog-qemu`: `scripts/smoke-catalog-target.sh` can attempt the target
  through the generic catalog QEMU helper.
- `debian-qemu`: the dedicated Debian QEMU smoke helper covers the target.
- `ubuntu-qemu`: the dedicated Ubuntu QEMU smoke helper covers the target.
- `mfsbsd-kboot-qemu`: the dedicated mfsBSD kboot QEMU smoke helper covers the
  target.
- `metadata-only`: no live or QEMU smoke helper explicitly covers the target.

Hosted catalog documents use the same schema and can optionally carry top-level
freshness metadata:

```json
{
  "schema_version": 1,
  "published_at": "2026-05-07T08:00:00Z",
  "expires_at": "2026-05-14T08:00:00Z",
  "targets": []
}
```

Bootup authenticates hosted catalog bytes before parsing them. Operators must
provide either `--catalog-sha256` with the expected SHA-256 digest, or
`--catalog-signature` plus `--catalog-public-key` for a detached Ed25519
signature over the raw catalog bytes. When both digest and signature trust are
configured, both checks must pass. Signature and public-key files may contain
raw bytes or hex-encoded bytes.

Hosted catalog freshness is operator-controlled. `expires_at` is always
enforced when present. `--catalog-require-freshness` requires either
`published_at` or `expires_at`, and `--catalog-max-age` rejects documents whose
`published_at` timestamp is older than the configured duration. The optional
`--catalog-cache` path is updated only after hosted bytes pass authentication,
freshness checks, parsing, and provider validation. `--catalog-cache-fallback`
allows that cache to be used after a fetch failure, but cached bytes go through
the same checks again before provider registration.

The document is data only. It selects concrete targets for provider code that is
already compiled into bootup; it cannot load provider plugins or executable
policy. `source.base_url` is an absolute HTTP(S) provider source root for that
target, and `source.iso_name` is a pathless installer ISO filename used by
providers that need one. The generic `linux` provider also accepts
`source.kernel_path`, optional `source.initrd_path`, and `source.cmdline`.
Those paths are clean relative URL paths resolved against `source.base_url`.
The command line may include `{base_url}` to refer to the trimmed source root.
Generic Linux source entries may also include `source.kernel_sha256` and
`source.initrd_sha256` as 64-character SHA-256 hex digests. When an initrd path
is present and either hash is supplied, both kernel and initrd hashes are
required. Pinned generic Linux artifacts are verified before staging, and the
catalog matrix reports the target as `hash-pinned`.
`lifecycle` is informational decoration for operator display; it is not
signature, checksum, transport, keyring, or other trust material.

Target options are non-executable catalog data. Supported option types are
`bool`, `enum`, and `string`. Boolean options append their `fragment` only when
selected as `true`. Enum options define allowed `values`, each with an optional
fragment. String options expand exactly one `{value}` placeholder in
`template`. Bootup rejects duplicate option IDs, unsupported types, invalid enum
values, and fragments or templates with surrounding whitespace or control
characters. Options are non-secret because selected values are expanded into
operator-visible command-line or loader-argument diagnostics. The `secret`
marker remains rejected; secrets use separate target declarations instead. See
[policy.md](policy.md).

Secret declarations describe provider-owned sensitive inputs. Each declaration
has an ID, label, purpose, requirement flag, and delivery mode. The first
delivery mode is `staged-file`: bootup validates the operator input from
`--secret id=/absolute/path`, keeps the value out of public boot plan fields,
and lets the provider stage a private `0600` file inside its staging tree.
Secret source paths and staged paths are diagnostics redaction values. The
default catalog currently declares no secret inputs, and no compiled distro
provider consumes a secret input yet; the provider-facing API is covered by
focused registry tests so a future provider can opt in without using target
options for secret material.

Operators select options with repeatable `--option id=value` flags. Selected
fragments are applied by boot action. Linux kexec targets append them after
provider defaults and before `--append-cmdline`, so operator append text remains
the final command-line addition. FreeBSD kboot targets append them to the
`loader.kboot` argument list because they do not boot through a Linux kernel
command line.

The `local` provider exposes `local-disk-auto` as a `localboot` action. It does
not download artifacts; handoff invokes u-root's local boot command to inspect
local boot configuration and continue from disk.

The `mfsbsd` provider exposes `mfsbsd-142-amd64` as a `freebsd-kboot` action.
It downloads and verifies the pinned mfsBSD 14.2 amd64 ISO, downloads and
verifies the pinned FreeBSD 15.0 `base.txz`, extracts `loader.kboot` from the
FreeBSD archive, extracts the mfsBSD ISO without mounting it from Linux, and
presents the extracted mfsBSD memory-root tree through `hostfs_root`. The
FreeBSD loader preloads `mfsroot`; after the kernel jump, the target mounts
`ufs:/dev/md0` and reaches the mfsBSD serial login. This is the supported BSD
rescue bridge for now. It is not equivalent to booting the stock FreeBSD
bootonly installer, because stock FreeBSD media still expects target-visible
`cd9660` root media after the kernel starts.

The target passes explicit mfsBSD runtime loader variables for auto-DHCP and
hostname. The stock image enables SSH and uses `root` with password `mfsroot`;
use that as a temporary rescue credential and rotate any installed-system
secrets separately. The catalog exposes a non-secret hostname option:

```sh
bootup --mode=boot-target --target=mfsbsd-142-amd64 \
  --option hostname=rescue-a
```

That option appends `mfsbsd.hostname=rescue-a` to the `loader.kboot` argument
list after the default `mfsbsd.hostname=mfsbsd` argument. Bootup does not expose
root password, password hash, or SSH key options yet. The secret input API
avoids loader-argument leakage, but mfsBSD would still need a provider-specific
consumer that safely modifies the memory-root image or another private delivery
point before those values can be used.

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
bootup --mode=discover-targets --discovery-family=fedora
```

The Debian provider discovers amd64 netboot installers from the configured
mirror, `discovery_url`, or optional local `discovery_file` metadata path. It
fetches the `dists/` index, filters release aliases such as `stable` and
`testing`, probes each release for amd64 netboot checksum metadata, and returns
one target per available release.

The Ubuntu provider discovers amd64 netboot installers from the configured
release index or optional local `discovery_file` metadata path. It reads
release links, checks each release `SHA256SUMS` file for a live-server amd64
ISO, probes the `netboot/amd64/linux` and `netboot/amd64/initrd` paths, and
returns concrete targets that the normal Ubuntu planner can stage.

The Fedora provider discovers amd64 Server netboot installers from the
configured releases index or optional local `discovery_file` metadata path. It
filters numeric release directories, probes each candidate for
`Server/x86_64/os/images/pxeboot/vmlinuz` and
`Server/x86_64/os/images/pxeboot/initrd.img`, and returns concrete targets that
the normal Fedora planner can stage. Static Fedora targets continue to resolve
`images/pxeboot/vmlinuz`, `images/pxeboot/initrd.img`, and an `inst.repo=`
command line from the target source URL or an operator-supplied `release_url`
override.

Fedora planning uses hash precedence of provider runtime pins, then target
source pins, then install-tree `.treeinfo`. The embedded Fedora static catalog
stores `.treeinfo` SHA-256 pins for its pxeboot artifacts so dry-run planning
and catalog matrix output stay hermetic. Discovered or custom Fedora targets
without pins fetch `.treeinfo` over HTTPS and fail planning unless both
`images/pxeboot/vmlinuz` and `images/pxeboot/initrd.img` have SHA-256 entries.

The generic Linux provider currently handles openSUSE Leap, Arch Linux, and
GParted Live catalog targets. These are Linux-shaped paths: kernel plus
optional initrd plus command line, staged over HTTPS.

MemTest86+ 8.00 is intentionally not in the default catalog. Its x86_64 image
boots when a firmware or bootloader enters the Linux boot protocol directly,
but it does not satisfy the currently implemented Linux kexec handoff paths.
It should be reintroduced only with a dedicated bootloader-style handoff, a
known kexec-compatible image, or a proven real-mode `kexec_load` loader.

Discovery is timeout-bound and explicit. Providers accept optional
`discovery_url`, `discovery_file`, `discovery_timeout`, and lifecycle
decoration in `--provider-config`; Debian, Fedora, and Ubuntu use the same
lifecycle map shape keyed by release. Local discovery metadata is used only to
find candidates; discovered target source URLs still use the configured
HTTP(S) provider source. If the primary discovery index fails, the
already-loaded static catalog targets remain available. If one candidate's
optional metadata or probe fails, providers skip that candidate and continue
checking the rest.

Discovery results can carry lifecycle decoration such as `supported`,
`obsolete`, `eol`, or `unknown`. That metadata is displayed to operators as
information only. Providers must not treat lifecycle metadata as signature,
checksum, keyring, transport, or other trust material.

## Future mode: broader distro discovery

Provider discovery can expand beyond the current Debian, Fedora, and Ubuntu
amd64 netboot targets to additional distributions, architectures, variants,
install options, and optional lifecycle data sources. That logic remains
outside the static catalog contract so static catalog documents stay stable
concrete target lists.

Stock BSD installers, HDT, memdisk ISO images, syslinux COM32 modules, and iPXE
chainload flows remain intentionally deferred unless they fit an implemented
handoff. The supported BSD-adjacent exception is the mfsBSD memory-root target
above, which uses FreeBSD `loader.kboot` and does not require target-visible
root media. The salstar BSD and several tool paths depend on bootloader
semantics that are not the same as Linux kernel/initrd kexec. u-root's
Multiboot support helps only for payloads that are actually
Multiboot-compatible; stock FreeBSD 15.0 release artifacts still need a
FreeBSD loader plus target-visible root media, EFI chainload, disk/ISO
chainload, or another dedicated handoff. OpenBSD installer media is in the same
deferred class: `bsd.rd` is a useful ramdisk installer/recovery kernel, but the
supported OpenBSD paths load it through OpenBSD boot blocks, `boot`, `cdboot`,
`pxeboot`, or EFI/BIOS media rather than through Linux kexec or FreeBSD
`loader.kboot`. Add those targets only after bootup has a proven executor
family for their handoff type.

## Implemented mode: signed local dynamic policy

Signed local dynamic policy can select an already-known target before provider
planning. The policy result is data only: target ID, selected non-secret
options, and optional secret references that map target-declared secret IDs to
operator-provided secret input IDs. Bootup validates the selected target,
options, and secret references against the current static catalog inventory
before staging artifacts.

Use `--mode=policy-target` to resolve and print the policy-selected boot plan,
or use the same policy flags with `plan-target`, `stage-target`, or
`boot-target` to let policy supply the target for those non-interactive modes.
Policy documents must be signed with an operator-supplied Ed25519 public key
and must carry freshness metadata. Cache fallback is local and authenticated;
remote policy URLs, executable policy engines, plugins, and provider-defined
runtime code are still deferred. The policy boundary is documented in
[policy.md](policy.md).
