## MODIFIED Requirements

### Requirement: Catalog target live smoke selection
Bootup SHALL provide an explicit opt-in live smoke path for selected static
catalog targets that can be exercised by currently implemented boot actions,
and SHALL use the catalog smoke coverage classification when deciding whether a
target is supported by the live catalog staging path.

#### Scenario: Supported target is smoke selectable
- **WHEN** a static catalog target is classified for live catalog staging smoke
  support
- **THEN** the live smoke path SHALL allow the target to be selected by target
  ID

#### Scenario: Unsupported action is skipped
- **WHEN** a static catalog target lacks live catalog staging smoke support,
  including targets that require memdisk, syslinux COM32, HDT, BSD-specific
  handoff, chainload, local boot, or a dedicated non-catalog smoke helper
- **THEN** the live smoke path SHALL report that the target is unsupported
  rather than attempting to boot it
