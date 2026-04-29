## Context

Bootup currently uses `TextMenu` for both `list-targets` and interactive menu
mode. That is robust on serial consoles, but it cannot offer animated feedback
or a visually distinct target picker. Bubble Tea, Bubbles, and Lip Gloss are
already present in the dependency graph through the u-root applets, so this
change can make them direct dependencies without adding a new family of code.

## Goals / Non-Goals

**Goals:**

- Make `--mode=menu` visually bold on capable terminals.
- Preserve the plain 80-column text path for non-interactive use and simple
  consoles.
- Keep provider planning, staging, verification, and kexec behavior unchanged.
- Make the interactive selection model testable without requiring a TTY.

**Non-Goals:**

- No framebuffer UI in this change.
- No mouse support, network provider changes, or verification changes.
- No runtime plugin loading.

## Decisions

- Add a separate `RichMenu` beside `TextMenu`.
  - Rationale: the current text renderer stays simple and remains available
    for automation and fallback.
  - Alternative: replace `TextMenu` entirely. That would make tests and
    redirected input less predictable.

- Use Bubble Tea for the interactive target picker.
  - Rationale: it gives a model/update/view split, keyboard handling, and a
    straightforward path to spinners and progress components.
  - Alternative: hand-roll ANSI rendering. That would duplicate terminal state
    management and be harder to test cleanly.

- Gate rich mode on stdin/stdout being terminal files.
  - Rationale: Bubble Tea wants an interactive terminal; redirected input
    should keep the existing `target> ` prompt.
  - Alternative: add a required flag. Auto-detection gives the right behavior
    for the initramfs default while preserving scripts.

- Keep the first Bubble Tea model small and message-driven.
  - Rationale: `Update` and `View` need to remain fast, state changes should
    happen through the normal message flow, and dimensions should be explicit
    because terminal layout arithmetic is easy to break.
  - Reference: https://leg100.github.io/en/posts/building-bubbletea-programs/

## Risks / Trade-offs

- Terminal escape handling can vary on firmware KVMs and serial concentrators.
  Mitigation: keep plain fallback and expose an explicit UI selector.

- Animation can be distracting or expensive on slow serial links.
  Mitigation: keep frame rates modest and use compact views that fit 80x25.

- Adding direct Bubble Tea usage increases the bootup binary dependency
  surface.
  Mitigation: dependencies are already needed by the u-root build path; tests
  cover fallback behavior if rich mode is unavailable.
