## MODIFIED Requirements

### Requirement: Discovery failure handling
Bootup SHALL report dynamic discovery failures without corrupting the static
catalog target list.

#### Scenario: Discovery source fails
- **WHEN** a provider discovery source is unavailable, malformed, or times out
- **THEN** bootup SHALL report the discovery failure to the operator and keep
  the current stage-1 environment available for diagnosis

#### Scenario: Discovery candidate probe fails
- **WHEN** dynamic discovery can read the primary discovery source but one
  candidate release probe is unavailable or returns an unexpected status
- **THEN** bootup SHALL skip that candidate and continue evaluating other
  candidate releases

#### Scenario: Discovery finds no targets
- **WHEN** dynamic discovery succeeds but returns no concrete targets
- **THEN** bootup SHALL report the empty result clearly and keep the current
  stage-1 environment available for diagnosis

#### Scenario: Static catalog remains available
- **WHEN** dynamic discovery fails for one provider family
- **THEN** bootup SHALL preserve already-loaded static catalog targets and other
  provider families

### Requirement: Shared provider discovery HTTP helpers
Bootup SHALL keep common provider discovery HTTP request behavior in shared,
tested helper code.

#### Scenario: Provider probes optional discovery artifacts
- **WHEN** a provider probes a candidate discovery URL
- **THEN** shared helper behavior SHALL treat HTTP 404 and missing local files
  as absence, report other unexpected statuses, and support GET fallback when
  HEAD is not allowed

#### Scenario: Provider fetches discovery metadata
- **WHEN** a provider fetches discovery metadata through the shared helper
- **THEN** the helper SHALL bind the request to the caller context and return
  response status separately from the response body

#### Scenario: Provider fetches local discovery metadata
- **WHEN** provider discovery metadata is configured with a local metadata path
- **THEN** the shared helper SHALL read that local file or directory index and
  return HTTP-like success or not-found status to the provider
