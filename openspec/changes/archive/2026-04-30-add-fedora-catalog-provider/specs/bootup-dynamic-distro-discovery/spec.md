## ADDED Requirements

### Requirement: Shared provider discovery HTTP helpers
Bootup SHALL keep common provider discovery HTTP request behavior in shared,
tested helper code.

#### Scenario: Provider probes optional discovery artifacts
- **WHEN** a provider probes a candidate discovery URL
- **THEN** shared helper behavior SHALL treat HTTP 404 as absence, report other
  unexpected statuses, and support GET fallback when HEAD is not allowed

#### Scenario: Provider fetches discovery metadata
- **WHEN** a provider fetches discovery metadata through the shared helper
- **THEN** the helper SHALL bind the request to the caller context and return
  response status separately from the response body
