## 1. Metadata Parsing

- [x] 1.1 Add failing Fedora provider tests for `.treeinfo` checksum parsing and malformed or missing checksum entries.
- [x] 1.2 Implement dependency-free `.treeinfo` checksum extraction for required pxeboot artifacts.

## 2. Fedora Planning

- [x] 2.1 Add failing Fedora planning tests for metadata-backed default hashes, fail-closed metadata errors, and explicit runtime pin override.
- [x] 2.2 Wire `.treeinfo` fetching into Fedora planning when runtime pins are absent.

## 3. Documentation and Verification

- [x] 3.1 Document Fedora `.treeinfo` checksum posture in launch, providers, and security docs.
- [x] 3.2 Run required Go, lint, OpenSpec, and diff checks.
