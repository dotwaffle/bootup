## Why

Bootup now has a usable provider catalog and two discovery-capable providers,
but the default distro set is still narrow and the embedded catalog is still
hand-maintained JSON. The next useful step is to add one RHEL-family provider,
make static catalog growth less error-prone, clarify hosted catalog security
requirements, and extract duplicated discovery HTTP helpers before adding more
providers.

## What Changes

- Add a compiled-in Fedora Server amd64 netboot provider and default static
  catalog entries for supported Fedora Server releases.
- Generate the embedded static catalog from a structured source file instead
  of editing `default.json` directly.
- Allow the generated catalog source to carry static lifecycle decoration so
  lifecycle metadata has a concrete local source before any external service
  integration.
- Refactor common provider HTTP discovery helpers used by Debian and Ubuntu.
- Document hosted static catalog authenticity/freshness requirements without
  implementing runtime URL catalog loading.
- Add tests and smoke coverage for the generated catalog, Fedora planning and
  staging, lifecycle decoration, shared helper behavior, and default catalog
  listing.

## Impact

- Affects provider registration, provider runtime config, static catalog data,
  generated catalog tooling, docs, tests, and OpenSpec specs.
- Does not add Arch, dynamic policy scripts, hosted catalog URL fetching,
  runtime provider plugins, release tags, or distro keyrings.
