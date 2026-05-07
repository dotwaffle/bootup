## 1. Implementation

- [x] 1.1 Add local file support to shared provider discovery fetch/probe helpers.
- [x] 1.2 Add provider runtime config parsing and validation for `discovery_file`.
- [x] 1.3 Wire `discovery_file` through compiled-in Debian, Ubuntu, and Fedora providers.
- [x] 1.4 Keep discovered target source URLs on configured HTTP(S) provider sources when local metadata is used.
- [x] 1.5 Continue provider discovery past per-candidate metadata/probe failures.
- [x] 1.6 Document local discovery metadata usage and failure behavior.

## 2. Verification

- [x] 2.1 Add tests for file-backed provider HTTP helpers.
- [x] 2.2 Add tests for provider config `discovery_file` validation.
- [x] 2.3 Add provider discovery tests for local metadata and skipped candidate failures.
- [x] 2.4 Run OpenSpec validation and relevant Go/lint checks.
