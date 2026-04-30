## Context

The default bootup release payload must not embed distribution-specific trust
bundles. Today the Debian provider can only become staging-capable in the
default command path through generated compile-time trust source, while the
Ubuntu provider already accepts caller-supplied release key material and
artifact hashes but has no CLI-facing configuration path.

Provider code is compiled in at build time. Runtime provider loading and a full
provider catalog are intentionally separate concerns, so this design only
configures the providers already present in the binary.

## Goals / Non-Goals

**Goals:**

- Provide an operator runtime path for provider source and verification inputs.
- Keep trust material external to default release artifacts.
- Fail fast on invalid provider configuration before target discovery or
  artifact retrieval.
- Preserve default no-config target discovery behavior while requiring runtime
  configuration for provider-specific trust material.

**Non-Goals:**

- No release tags or release signing changes.
- No runtime provider plugin loading.
- No broad provider catalog schema.
- No committed distribution keyrings, generated trust source, or binary payloads.

## Decisions

### Use a JSON file loaded by `--provider-config`

The bootup command will accept `--provider-config <path>` and parse a small
JSON document. JSON avoids adding YAML/TOML dependencies to the stage-1 binary
and is easy to generate from provisioning systems.

Alternative: add many CLI flags or environment variables. That keeps parsing
simple for two providers, but it scales poorly as provider count grows and
would make per-provider trust policy harder to audit.

### Key entries by provider ID

The file will contain a `providers` object keyed by compiled-in provider IDs,
for example `debian` and `ubuntu`. Unknown provider IDs fail startup so typos do
not silently disable verification.

Alternative: create a target-level catalog now. That will likely be needed
later, but the current requirement is to configure verification for compiled-in
providers and should not force the catalog model early.

### Read trust material from file paths

Provider config entries reference local keyring files by path. The command reads
those files at startup and passes bytes to provider constructors. The config file
does not inline keyring material, keeping logs and config review easier.

Alternative: inline base64 keyring blobs. That is self-contained but makes the
operator config harder to inspect and increases the chance of accidental
copy/paste or log exposure.

### Remove the Debian compile-time trust hook from default registration

For Debian, archive trust material comes from runtime provider configuration in
the default command path. Local self-contained initramfs builds include the
operator-selected keyring as an initramfs file and point provider config at that
path. The default release build embeds no distro trust.

Alternative: keep the generated Go source hook as a fallback. That preserves one
legacy local build path, but it keeps a Debian-specific trust-material carve-out
in the command wiring and does not scale to many providers.

## Risks / Trade-offs

- Provider-specific fields remain in the first config version -> isolate parsing
  in a small command-facing package so later providers can add typed entries
  without changing provider internals.
- JSON is less friendly for hand-written comments -> document minimal examples
  and keep the schema shallow.
- Unknown provider IDs fail startup -> this is stricter than ignoring unknown
  entries, but it prevents operators from believing verification is configured
  when it is not.
- Relative keyring paths depend on process working directory -> document that
  operators should use absolute paths in initramfs or ISO environments.
