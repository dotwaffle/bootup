## MODIFIED Requirements

### Requirement: Failure diagnostics bundle
Bootup SHALL provide an opt-in diagnostics directory that records failure
context for startup modes without changing normal console output.

#### Scenario: Diagnostics are disabled by default
- **WHEN** bootup starts without diagnostics configuration
- **THEN** it SHALL preserve existing stdout, stderr, logging, and error
  behavior without writing a diagnostics bundle

#### Scenario: Failed run writes diagnostics
- **WHEN** bootup starts with a diagnostics directory and the selected mode
  fails after flag parsing
- **THEN** it SHALL write a diagnostics bundle containing a JSON summary,
  captured stdout text, and captured stderr/log text

#### Scenario: Diagnostic summary avoids secret values
- **WHEN** bootup writes a diagnostics summary
- **THEN** the summary SHALL include mode, target ID, discovery family ID,
  selected option IDs, catalog source posture, provider config path presence,
  policy source kind, policy cache posture, policy fallback category, and final
  error, and MUST NOT include selected option values, provider config contents,
  policy response bodies, trust material bytes, passwords, tokens, or SSH keys

#### Scenario: Diagnostics failure preserves original error
- **WHEN** bootup fails and diagnostics bundle writing also fails
- **THEN** bootup SHALL return the original boot failure while reporting the
  diagnostics write failure as secondary context
