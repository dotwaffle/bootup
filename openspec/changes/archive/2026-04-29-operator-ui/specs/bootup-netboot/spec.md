## MODIFIED Requirements

### Requirement: Serial-first operator interface
Bootup SHALL provide an operator interface suitable for serial consoles, IPMI
consoles, and console-like KVM sessions.

#### Scenario: Serial console is available
- **WHEN** bootup starts with a serial console
- **THEN** bootup SHALL render target selection and failure messages in an
  interface that remains usable in an 80x25 viewport

#### Scenario: Rich terminal is available
- **WHEN** bootup menu mode starts with interactive terminal input and output
- **THEN** bootup SHALL render a bold target picker with color, keyboard
  navigation, animated activity, and selected-target detail

#### Scenario: Rich terminal is unavailable
- **WHEN** bootup menu mode starts with redirected input, redirected output, or
  an incompatible terminal
- **THEN** bootup SHALL fall back to the plain text target prompt

#### Scenario: Operator chooses a target by index
- **WHEN** bootup renders the serial menu
- **THEN** each target SHALL have a stable numeric index for selection

#### Scenario: Operator navigates rich menu
- **WHEN** bootup renders the rich terminal menu
- **THEN** arrow keys, `j`, and `k` SHALL move the highlighted target and
  Enter SHALL select it

#### Scenario: Boot progress is visible
- **WHEN** bootup is planning, staging, verifying, or loading a selected target
- **THEN** the serial interface SHALL render a concise status message that fits
  inside an 80-column viewport

#### Scenario: Rich boot progress is visible
- **WHEN** bootup is planning, staging, verifying, or loading a selected target
  after rich-menu selection
- **THEN** bootup SHALL render visually distinct progress status with animated
  activity before handoff

#### Scenario: Boot failure is visible
- **WHEN** planning, verification, staging, or kexec fails
- **THEN** the serial interface SHALL render a readable fatal error and keep the
  current environment available for diagnosis

#### Scenario: Framebuffer is unavailable
- **WHEN** bootup cannot use a framebuffer display
- **THEN** bootup SHALL still allow target selection and boot execution through
  the operator interface
