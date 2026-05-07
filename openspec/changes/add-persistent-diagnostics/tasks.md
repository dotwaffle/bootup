## 1. Diagnostics Core

- [x] 1.1 Add failing tests for diagnostics bundle writing, summary redaction, and secondary write errors.
- [x] 1.2 Implement the diagnostics bundle writer and redacted summary model.

## 2. CLI Integration

- [x] 2.1 Add failing CLI tests for `--diagnostics-dir` failure capture and default no-op behavior.
- [x] 2.2 Wire `--diagnostics-dir` through startup, stdout/stderr capture, and app logging.

## 3. Documentation and Verification

- [x] 3.1 Document diagnostics usage for local and VM runs.
- [x] 3.2 Run required Go, lint, OpenSpec, and diff checks.
