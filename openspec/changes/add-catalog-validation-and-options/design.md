## Context

Bootup now has an action-based static catalog with generic Linux and local boot
targets. The next risk is operational confidence: catalog entries must be
proved against a VM where practical, installer knobs need to be data-driven, and
operators need enough catalog detail to choose a target without reading JSON.

The current executor set can cover Linux kexec and local boot. BSD, memdisk,
syslinux COM32, HDT, and chainload-style targets remain out of scope until a
dedicated executor design exists.

## Goals / Non-Goals

**Goals:**

- Add tagged live smoke tests for selected static catalog targets.
- Introduce a small, declarative option model for catalog targets.
- Apply selected options as validated command-line additions during planning.
- Improve catalog list/show output without requiring dynamic discovery.

**Non-Goals:**

- Runtime URL-hosted catalogs.
- Dynamic release discovery or EOL lookup.
- Distro-specific signature or keyring policy.
- New non-Linux executors for BSD, memdisk, syslinux, HDT, or chainload.

## Decisions

1. Smoke tests stay opt-in and target-specific.

   Live tests will remain behind explicit tags or environment variables because
   they require QEMU, host kernel support, network access, and time. The first
   coverage should exercise one kernel+initrd generic Linux target. MemTest86+
   8.00 was removed from the default catalog after smoke validation showed that
   the image boots through firmware-style Linux boot protocol entry but not
   through bootup's current kexec paths.

2. Options are catalog data, not provider-specific code.

   Each target can declare option definitions with an ID, label, type, allowed
   values where relevant, and command-line fragments. Providers receive already
   selected options through planning input and merge them into the boot plan.
   This keeps salstar-style options such as serial console, VNC install, text
   install, mirror URL, or automated install file URL out of provider logic
   unless a provider truly needs custom planning behavior later.

3. Option command-line expansion is conservative.

   Initial option types should be boolean and enum/string templates. Values must
   be validated before expansion, generated fragments must not contain leading
   or trailing whitespace, and final command-line append order must be stable:
   provider defaults, catalog option fragments, then operator
   `--append-cmdline`.

4. Catalog discovery output is operator-oriented.

   `list` should remain compact and scannable. `show` should provide the full
   target metadata, artifact references, boot action, lifecycle decoration, and
   option definitions. JSON output can be added where the existing command style
   already supports structured output; otherwise keep this change to stable text
   output and tests.

## Risks / Trade-offs

- Live smoke tests can be flaky because they depend on QEMU, network, and
  upstream mirrors -> keep them explicitly gated and document requirements.
- Command-line options can become a second scripting language -> keep the first
  model limited to validated fragments and templates.
- Catalog output changes can break scripts if existing text is parsed -> prefer
  adding fields to detailed output and keep compact list columns predictable.
- Some distro options need more than cmdline fragments -> defer provider-specific
  option handlers until a concrete target needs them.
