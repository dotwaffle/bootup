## 1. Implementation

- [x] 1.1 Add a catalog composition helper that appends validated documents and rejects duplicate target IDs.
- [x] 1.2 Add `--catalog-include-default` to CLI catalog loading.
- [x] 1.3 Preserve existing replacement behavior when composition is not requested.
- [x] 1.4 Document additive composition and duplicate-ID rejection.

## 2. Verification

- [x] 2.1 Add unit tests for catalog composition and duplicate target rejection.
- [x] 2.2 Add CLI tests for local catalog composition and replacement compatibility.
- [x] 2.3 Run OpenSpec validation and relevant Go/lint checks.
