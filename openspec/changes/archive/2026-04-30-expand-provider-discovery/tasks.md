## 1. Proposal And Wording

- [x] 1.1 Refresh README/docs/spec wording so provider discovery is implemented, not deferred.
- [x] 1.2 Document hosted static catalog authenticity and freshness requirements without adding URL loading.

## 2. Runtime Config And Lifecycle

- [x] 2.1 Add provider runtime config fields for discovery URL, discovery timeout, and lifecycle maps.
- [x] 2.2 Pass discovery and lifecycle config into Debian and Ubuntu providers.
- [x] 2.3 Validate lifecycle status/date and discovery duration errors before provider registration.

## 3. Ubuntu Discovery

- [x] 3.1 Add Ubuntu discovery family support without eager discovery.
- [x] 3.2 Discover Ubuntu amd64 netboot targets from a configured release index.
- [x] 3.3 Allow Ubuntu discovered targets to plan through the normal provider path.
- [x] 3.4 Add timeout-bound Ubuntu discovery failure handling.

## 4. Discovery UX

- [x] 4.1 Improve empty discovery diagnostics in non-interactive and menu flows.
- [x] 4.2 Keep static target listing available when configured discovery fails.

## 5. Static Catalog Expansion

- [x] 5.1 Add Debian forky amd64 netboot to the embedded static catalog.
- [x] 5.2 Update catalog tests, docs, and VM expectations for the added target.

## 6. Validation

- [x] 6.1 Add unit and fixture-tag coverage for provider discovery config, lifecycle decoration, Ubuntu discovery, and discovered target planning.
- [x] 6.2 Run lint, Go tests, tagged tests, build, OpenSpec validation, archive, commit, push, and check CI.
