## Context

The Fedora provider currently plans `images/pxeboot/vmlinuz` and
`images/pxeboot/initrd.img` from the selected Server install-tree URL. If the
operator supplies explicit runtime hash pins, staging verifies both artifacts;
otherwise staging only requires HTTPS transport. Fedora Server install trees
also publish `.treeinfo` metadata whose `[checksums]` section includes SHA-256
entries for both pxeboot artifacts. Bootup's target source metadata can also
carry per-target kernel and initrd hash pins.

## Goals / Non-Goals

**Goals:**

- Use Fedora install-tree `.treeinfo` checksums for Fedora netboot planning
  when explicit runtime and target source hash pins are absent.
- Preserve Fedora pxeboot hash pins in the embedded catalog so default catalog
  matrix and dry-run planning stay hermetic.
- Keep explicit runtime hash pins authoritative over target source pins and
  avoid metadata fetches in either pinned path.
- Fail closed when default Fedora metadata cannot authenticate both required
  pxeboot artifacts.
- Keep tests hermetic and avoid adding Fedora keyrings or committed checksum
  payloads.

**Non-Goals:**

- Add Fedora GPG verification or embed Fedora release keys.
- Add live Fedora smoke tests or depend on upstream network state in CI.
- Change generic Linux, Ubuntu, Debian, or catalog trust semantics.

## Decisions

- Use checksum precedence of runtime pins, then target source pins, then
  `.treeinfo`. Runtime pins are explicit operator policy, target source pins
  keep the embedded catalog hermetic, and `.treeinfo` covers discovered or
  custom Fedora targets without precomputed catalog hashes.
- Fetch `.treeinfo` during Fedora planning only when runtime and target source
  pins are absent. Planning already receives a context and provider HTTP
  client, and the resulting `provider.BootPlan` already has SHA-256 fields
  consumed by staging.
- Parse only the `[checksums]` section needed by bootup. The file is INI-like,
  but the required shape is simple: `path = sha256:<hex>`. A small parser keeps
  the change dependency-free and avoids treating unrelated sections as trust
  material.
- Require both `images/pxeboot/vmlinuz` and `images/pxeboot/initrd.img`
  SHA-256 entries. Missing, malformed, or non-SHA-256 entries fail planning
  before any artifact is staged.
- Keep explicit provider runtime pins as the strongest local policy. If pins
  are configured, planning uses them and does not fetch `.treeinfo`, preserving
  offline/pinned operator workflows.

## Risks / Trade-offs

- Fedora planning without any pins becomes network-bound -> use the existing
  context/client path and keep runtime or target source pins as offline escape
  hatches.
- `.treeinfo` over HTTPS is still weaker than signed metadata -> document the
  posture as metadata-backed HTTPS integrity, not a Fedora signature chain.
- Parser drift if Fedora changes metadata shape -> limit parsing to the
  documented entries and fail closed when the expected section or values are
  absent.
