## Context

Bootup validates static catalog documents before registering provider targets,
and the default tests already exercise listing, selecting, and selected live
smoke paths. The missing piece is a catalog-wide view that says whether each
registered target can be planned and which smoke helper, if any, covers that
target.

Provider `Plan` methods are currently dry-run operations: they assemble URLs,
kernel command lines, verification metadata, and action-specific plan fields
without downloading artifacts. Provider `Stage` methods perform network I/O and
verification. The matrix must stay on the `Plan` side of that boundary.

## Goals / Non-Goals

**Goals:**
- Render a stable target-by-target catalog matrix for operators and tests.
- Classify plan status, artifact trust posture, and smoke coverage without
  contacting mirrors or launching QEMU.
- Reuse the same smoke classification in live tests and operator output.
- Make plan failures visible enough for CI/operator checks.

**Non-Goals:**
- Do not add artifact downloads, staging, QEMU execution, or live network
  checks to the default suite.
- Do not treat hosted catalog trust as boot-artifact trust.
- Do not unblock deferred memdisk, stock FreeBSD installer, or OpenBSD routes.

## Decisions

- Add an internal catalog conformance report rather than embedding this logic
  only in tests. This gives operators the same view that tests assert and keeps
  the classification independent of a specific CLI renderer.

- Add `--mode=catalog-matrix` rather than overloading `list-targets`. The list
  output remains compact and human-oriented; the matrix can be tabular and
  assertion-friendly without changing existing list expectations.

- Classify trust from the dry-run boot plan. Pinned artifact hashes, signed
  metadata, release metadata, HTTPS-only artifacts, local boot, and
  unverified/partial states are distinct enough to guide follow-up hardening
  without making new security promises.

- Treat smoke coverage as explicit helper support. Generic Linux catalog
  targets can use the live catalog staging and catalog QEMU helpers, Debian and
  Ubuntu keep their dedicated QEMU smoke labels, and mfsBSD keeps its kboot
  QEMU label.

## Risks / Trade-offs

- Plan dry-runs can miss Stage-time verification problems. Mitigation: report
  this as a conformance matrix, not as proof that live staging or boot succeeds.

- A target may become live-smokeable before the classifier is updated.
  Mitigation: keep the classifier table small and covered by tests so new
  helper support has an obvious update point.

- The matrix may expose provider configuration differences, such as runtime
  hash pins. Mitigation: derive trust from the actual configured registry and
  do not claim that catalog data alone supplies provider trust.
