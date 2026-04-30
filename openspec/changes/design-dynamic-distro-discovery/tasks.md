## 1. Provider Discovery Model

- [ ] 1.1 Add provider discovery interfaces and target lifecycle metadata types.
- [ ] 1.2 Add registry support for listing discovery families without running discovery.
- [ ] 1.3 Add registry support for running discovery for one selected provider family.

## 2. Operator Flow

- [ ] 2.1 Add app/UI flow for selecting a discovery family and then a discovered target.
- [ ] 2.2 Add non-interactive discovery diagnostics for listing discovered targets.
- [ ] 2.3 Render lifecycle decoration without treating it as trust material.

## 3. Provider Implementation

- [ ] 3.1 Implement one provider discovery path behind explicit tests.
- [ ] 3.2 Add timeout-bound failure handling for discovery source errors.
- [ ] 3.3 Keep static catalog target listing available when discovery fails.

## 4. Validation

- [ ] 4.1 Add unit and VM/integration coverage for discovery family listing and discovered target planning.
- [ ] 4.2 Update docs and archive the change after implementation.
