## Context

The release builder already gives each artifact a release-qualified filename and
writes a manifest with the release version, git commit, artifact sizes, and
hashes. The bootup binary itself does not expose this information, so an
operator who has only a copied binary or booted initramfs cannot ask the binary
which release metadata it carries.

## Goals / Non-Goals

**Goals:**

- Provide a stable `bootup --version` diagnostic path that does not initialize
  providers, catalogs, networking, or boot modes.
- Stamp the release build with the same release version and commit recorded in
  the release manifest, plus build date and source tree state.
- Make release validation compare the manifest metadata with the binary's own
  reported build metadata.

**Non-Goals:**

- Add runtime update checks, release signing, SBOM generation, or multi-arch
  release output.
- Change provider behavior or boot target planning.
- Make local development builds reproducible by default.

## Decisions

- Keep build metadata in a small internal package with linker-set variables.
  This avoids wiring globals through the app runtime and keeps the default
  development build behavior explicit.
- Format `--version` as a tab-separated diagnostic report. This matches the
  repository's existing operator-oriented command output and remains simple for
  the release validator to parse.
- Stamp release builds from `scripts/build-release.sh` with `-ldflags -X`
  values. The release workflow already centralizes artifact naming through this
  script, so the script is the narrowest place to keep filenames, manifest data,
  and binary metadata aligned.
- Record binary build metadata in a manifest object separate from the artifact
  list. Artifact entries remain focused on file identity and integrity while
  `bootupBuild` records executable metadata.

## Risks / Trade-offs

- Shell parsing of `--version` output could drift if the format changes. The
  release validator mitigates this by enforcing the field names used by the
  manifest comparison.
- Development builds will report fallback metadata. That is intentional; the
  release path is responsible for stamping publishable artifacts.
- Build date makes release binaries time-dependent. The script will honor
  `SOURCE_DATE_EPOCH` when present so controlled builds can provide a stable
  timestamp.
