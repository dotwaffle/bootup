## Context

The current rich menu has model-level tests and fallback tests, but it has not
yet been exercised through a pseudo-terminal. Bubble Tea depends on terminal
behavior for input handling, rendering, and terminal setup, so PTY tests are
the right next layer before more visual iteration.

## Goals / Non-Goals

**Goals:**

- Confirm `--mode=menu --ui=rich` can be driven through a PTY.
- Keep model `Update` and `View` fast and deterministic.
- Improve scanability without relying on non-serial-safe glyphs.
- Capture binary and initramfs size after the rich UI work.

**Non-Goals:**

- No framebuffer UI.
- No mouse controls.
- No live Debian or Ubuntu network smoke changes.

## Decisions

- Use `github.com/Netflix/go-expect` for PTY tests.
  - Rationale: it is already in the module graph and exposes a real terminal
    file for `term.IsTerminal` checks.

- Keep the QEMU smoke local and documented rather than part of the default
  suite.
  - Rationale: it depends on a local kernel, QEMU, and timing-sensitive serial
    output.

- Keep polish inside the existing `TargetPicker` model.
  - Rationale: the current UI is still small enough that a model tree would add
    complexity without enough payoff.

- Use Bubble Tea v2 import paths for application code.
  - Rationale: `charm.land/bubbletea/v2`, `charm.land/bubbles/v2`, and
    `charm.land/lipgloss/v2` are the current API surface. Older
    `github.com/charmbracelet/...` packages may remain indirect dependencies
    of u-root applets, but bootup's rich UI MUST NOT import them directly.

## Risks / Trade-offs

- PTY tests can be timing-sensitive.
  Mitigation: use explicit expect timeouts and short, stable output substrings.

- QEMU menu smoke may differ across host kernels and terminal environments.
  Mitigation: record the command and observed behavior in docs rather than
  making it a default gate.
