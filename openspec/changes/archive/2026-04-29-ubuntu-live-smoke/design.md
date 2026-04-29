## Context

The Ubuntu provider stages the official 26.04 netboot kernel and initrd over
HTTPS by default. That is intentionally weaker than Debian's signed metadata
chain, but it mirrors the common ISO download trust model and makes the target
usable now. We need a repeatable way to verify that the provider can stage
live artifacts and that the resulting kernel/initrd can be handed to kexec.

## Decisions

### Use the same QEMU networking fallback as Debian smoke

The local host kernel may not support `CONFIG_IP_PNP_DHCP` or built-in QEMU
NICs, so the smoke helper should use the same `e1000` module plus static QEMU
user-network addressing fallback as the Debian smoke helper.

### Keep Ubuntu smoke opt-in

Live Ubuntu checks require network and QEMU. Default `go test ./...` should
remain hermetic, with live checks gated by explicit environment variables.

### Treat timeout after installer boot as success for manual smoke

The smoke helper starts QEMU and can reach Ubuntu's installer environment. It
does not automate a full installer session. A timeout after kexec is acceptable
for manual smoke when the serial output shows the `[loading]` handoff and
target kernel boot.

## Risks / Trade-offs

- Ubuntu live mirrors or release paths can change; docs should state the
  exact target and command.
- HTTPS-only staging can catch transport failures but not provide the same
  archive-trust chain as Debian.
- The installer may rename NICs or require further kernel parameters after
  kexec; this smoke focuses on reaching the handoff.

## Rollout

1. Add the smoke script and live test.
2. Document commands and expected output.
3. Run standard tests, lint, and shell syntax checks.
