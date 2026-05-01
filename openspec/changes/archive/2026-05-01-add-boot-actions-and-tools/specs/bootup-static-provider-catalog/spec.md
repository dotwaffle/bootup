## ADDED Requirements

### Requirement: Target boot action metadata
Bootup SHALL validate optional boot action metadata on static provider targets.

#### Scenario: Target declares supported action
- **WHEN** a static target declares a supported boot action
- **THEN** bootup SHALL preserve that action in the target metadata passed to
  the selected provider

#### Scenario: Target distribution differs from provider
- **WHEN** a static target is handled by a generic compiled provider
- **THEN** bootup SHALL allow the target catalog distribution to name the
  target family rather than the compiled provider ID
