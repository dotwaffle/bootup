## Context

Fedora is a good next distro because its network installer follows the same
compiled-provider model bootup already uses: provider code can resolve a
concrete release tree into a kernel, initrd, and installer command line without
runtime plugins. Fedora's install docs show Server netboot artifacts under
`Server/x86_64/os/images/pxeboot/` and installation source selection through
`inst.repo=`.

## Design

### Fedora Provider

Add `internal/providers/fedora` as a compiled-in provider. Static catalog
targets use `architecture: "amd64"` for bootup consistency while Fedora source
URLs use `x86_64`. Each target's `source.base_url` points at a Fedora Server
install tree such as:

`https://download.fedoraproject.org/pub/fedora/linux/releases/44/Server/x86_64/os`

The provider plans:

- kernel: `<base>/images/pxeboot/vmlinuz`
- initrd: `<base>/images/pxeboot/initrd.img`
- cmdline: `inst.repo=<base> ip=dhcp console=ttyS0`

Staging follows the Ubuntu-style HTTPS-only default: when explicit kernel and
initrd hashes are absent, artifacts must be fetched over HTTPS. Optional
runtime SHA-256 pins can verify the netboot kernel and initrd. This keeps
Fedora trust material operator-configurable and avoids embedding distro
keyrings or adding a detached-signature-specific path.

### Catalog Generation

Add a compact source document under `internal/catalog/` and a Go generator that
emits the current schema-version-1 `default.json`. The generator expands common
target fields, keeps deterministic ordering, and preserves optional source and
lifecycle fields. `go generate ./internal/catalog` becomes the supported way to
refresh the embedded catalog after editing the source.

### Lifecycle Source

The generated catalog source can include lifecycle decoration per static
target. This is a local, static metadata source with explicit `source` strings,
not an online end-of-life lookup. Future work can replace or augment the source
data with generated data from endoflife.date or another configured service, but
this change keeps stage-1 behavior deterministic and offline.

### Shared Provider HTTP Helpers

Move duplicated HTTP fetch/status/probe/path helpers out of Debian and Ubuntu
into an internal provider helper package. Providers still own their discovery
logic and target semantics, but common request handling, 404-as-absence probes,
HEAD-to-GET fallback, and small URL path helpers are tested once.

### Hosted Catalog Design

Docs and specs continue to defer hosted catalog URL loading. The design
boundary becomes explicit: hosted catalogs require signed or pinned documents,
freshness semantics, cache policy, offline fallback behavior, and operator trust
configuration before bootup loads them at runtime.

## Risks / Trade-offs

- Fedora release URLs move to archive locations after EOL. Static catalog
  entries should track supported releases and can be updated by catalog source
  regeneration.
- HTTPS-only Fedora staging is weaker than signed artifact verification, but it
  matches the existing Ubuntu no-keyring default and keeps trust material an
  operator decision.
- The catalog generator adds one more tool path, so CI and docs must make it
  obvious when generated output is stale.
