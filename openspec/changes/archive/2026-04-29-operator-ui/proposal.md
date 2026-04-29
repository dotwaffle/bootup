## Why

Bootup currently exposes target selection through plain serial text, which is
usable but visually close to legacy PXE/iPXE menus. The MVP now has multiple
providers and real smoke coverage, so the operator-facing menu should become a
stronger first impression while staying practical on serial consoles.

## What Changes

- Add a rich terminal menu for `--mode=menu` when stdin/stdout are terminal
  files.
- Keep the existing plain text prompt as the fallback for redirected input,
  tests, and automation.
- Render bright provider cards, a bold bootup banner, keyboard navigation,
  an animated spinner, and progress-style status lines for planning,
  verification, staging, and loading.
- Add tests for target rendering, keyboard selection behavior, fallback mode,
  and failure output.

## Capabilities

### New Capabilities

### Modified Capabilities

- `bootup-netboot`: the serial-first operator interface gains a rich terminal
  mode while preserving plain serial fallback behavior.

## Impact

- Affects `internal/ui`, `internal/app`, `cmd/bootup`, and operator docs.
- Promotes existing Bubble Tea, Bubbles, and Lip Gloss dependencies to direct
  runtime dependencies for the bootup binary.
- Does not change provider APIs, verification hooks, staging behavior, or
  non-interactive modes.
