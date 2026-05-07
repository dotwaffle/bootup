## Context

The mfsBSD provider currently stages a verified mfsBSD ISO, extracts the memory
root without mounting the ISO, runs FreeBSD `loader.kboot`, and reaches a serial
login in smoke testing. The loader args contain the hostfs and serial pieces
needed for the handoff, but they do not explicitly carry mfsBSD's operator
runtime settings. Selected target options also only append Linux command-line
fragments, so `freebsd-kboot` targets cannot use catalog options yet.

Upstream mfsBSD images use `root` with password `mfsroot`, enable auto-DHCP via
`mfsbsd.autodhcp`, enable sshd, and include rescue packages such as `tmux`,
`rsync`, `smartmontools`, and `zfsinstall`. Bootup should expose those facts and
make the non-secret parts configurable.

## Goals / Non-Goals

**Goals:**

- Preserve the proven mfsBSD memory-root handoff.
- Pass explicit mfsBSD loader variables for auto-DHCP and hostname.
- Allow operators to override the mfsBSD hostname with a catalog target option.
- Apply selected target option fragments to `freebsd-kboot` loader arguments.
- Document login, DHCP, SSH, serial console, and basic rescue usage.

**Non-Goals:**

- Do not add root password, root password hash, SSH key, or other secret-bearing
  options in this change.
- Do not add OpenBSD or stock FreeBSD installer support.
- Do not vendor mfsBSD, FreeBSD, ISO, loader, initramfs, or VM artifacts.

## Decisions

1. Use loader arguments for runtime settings.

   `loader.kboot` already receives `key=value` arguments for hostfs, serial,
   and boot behavior. Adding `mfsbsd.autodhcp=YES` and
   `mfsbsd.hostname=mfsbsd` keeps data flow in the existing handoff path and
   matches mfsBSD's documented loader variable model.

2. Keep target options action-aware.

   `provider.ApplySelectedOptions` will continue appending fragments to
   `Cmdline` for Linux-shaped actions. For `freebsd-kboot`, it will append the
   same selected fragments to `FreeBSDKboot.Args`. Staging will prepend
   generated default loader args such as `hostfs_root` after the payload root is
   known, then preserve the selected operator fragments.

3. Limit the first option to hostname.

   Hostname is non-secret, useful in DHCP/SSH logs, and easy to validate through
   the existing string option path. Password customization is deliberately left
   out because stage and plan output currently print loader args; adding secret
   args before redaction would leak credentials.

## Risks / Trade-offs

- Duplicate loader variables may depend on loader ordering -> defaults are
  generated first and selected option fragments are appended after them so an
  explicit hostname option wins.
- Existing static catalog validation may reject malformed new option metadata
  -> tests and `go generate ./internal/catalog` cover generated catalog output.
- Operators may ask for password/SSH-key customization next -> document the
  current `root`/`mfsroot` path and explicitly defer secret-bearing knobs until
  bootup can avoid printing them.
