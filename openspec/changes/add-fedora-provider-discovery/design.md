## Context

The provider registry already supports discovery-capable providers. Debian and
Ubuntu implement `provider.Discoverer`; Fedora exposes static targets and can
plan Fedora Server netboot URLs from a release install tree.

## Goals / Non-Goals

**Goals:**
- Add a Fedora discovery family for amd64 Server netboot releases.
- Make the discovery base URL and timeout configurable.
- Return normal Fedora `provider.Target` values with source base URLs so the
  existing Fedora planner and stager handle selected targets.

**Non-Goals:**
- Do not scrape Fedora lifecycle or EOL state in this change.
- Do not add artifact hash discovery or signature policy.
- Do not contact Fedora mirrors in default tests.

## Decisions

- Discover from a Fedora releases index whose children are numeric release
  directories. Each candidate release maps to
  `<discovery>/<release>/Server/x86_64/os`.

- Probe both `images/pxeboot/vmlinuz` and `images/pxeboot/initrd.img` before
  returning a discovered target. This avoids listing releases that do not have
  the Server amd64 netboot shape the existing planner expects.

- Keep target IDs aligned with static Fedora IDs:
  `fedora-<release>-amd64-server-netboot`.

## Risks / Trade-offs

- Fedora mirror indexes can vary. Mitigation: make `discovery_url`
  configurable and keep parser/probe behavior covered by fixture tests.

- Discovery only covers Server x86_64 netboot. Mitigation: document the scope
  and leave other editions/architectures to future provider work.
