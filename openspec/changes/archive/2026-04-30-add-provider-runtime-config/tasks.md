## 1. Runtime Config Loader

- [x] 1.1 Add a hermetic parser test for valid Debian and Ubuntu provider entries.
- [x] 1.2 Add parser tests for malformed JSON, unknown providers, invalid hashes, and unreadable keyring paths.
- [x] 1.3 Implement the provider runtime config loader using the Go standard library.

## 2. Provider Wiring

- [x] 2.1 Add command/provider registration tests proving runtime config overrides provider defaults.
- [x] 2.2 Add `--provider-config` startup wiring that loads config before provider discovery.
- [x] 2.3 Apply Debian keyring/mirror config and Ubuntu release URL/keyring/hash config during default provider registration.
- [x] 2.4 Remove the Debian compile-time trust hook from default provider registration and local helper flows.

## 3. Documentation And Validation

- [x] 3.1 Document the provider runtime config file in launch and security docs.
- [x] 3.2 Run Go tests, linting, and OpenSpec validation.
- [x] 3.3 Archive the completed OpenSpec change after implementation.
