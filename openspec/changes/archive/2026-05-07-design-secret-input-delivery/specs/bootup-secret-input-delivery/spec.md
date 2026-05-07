## ADDED Requirements

### Requirement: Secret input declarations
Bootup SHALL model secret-bearing provider inputs separately from target
options and boot argument fragments.

#### Scenario: Target declares a required secret
- **WHEN** a target or compiled-in provider declares a required secret input
- **THEN** bootup SHALL expose the secret ID, label, purpose, and requirement
  status without exposing or embedding a secret value

#### Scenario: Target option attempts secret delivery
- **WHEN** a catalog target option attempts to deliver a secret through a
  command-line or loader-argument fragment
- **THEN** bootup SHALL reject that target before planning, staging, or handoff

### Requirement: File-backed secret inputs
Bootup SHALL accept secret values only through explicit local file-backed
operator inputs in the first secret delivery implementation.

#### Scenario: Required secret file is supplied
- **WHEN** an operator supplies a local file path for a required secret ID
- **THEN** bootup SHALL validate that the path is absolute, local, readable,
  regular, size-limited, and acceptable under the configured file-permission
  policy before provider planning

#### Scenario: Required secret is missing
- **WHEN** a required secret declaration has no operator-supplied value
- **THEN** bootup SHALL fail before provider planning or artifact staging

#### Scenario: Secret value is supplied inline
- **WHEN** an operator or catalog attempts to provide a secret value inline,
  through a target option, provider runtime config value, environment value, or
  command-line fragment
- **THEN** bootup SHALL reject that input before provider planning

### Requirement: Secret redaction
Bootup SHALL avoid printing, persisting, hashing, or otherwise exposing secret
values and secret source paths in normal output or diagnostics.

#### Scenario: Diagnostics include secret context
- **WHEN** bootup writes diagnostics for a run that references secret inputs
- **THEN** diagnostics MAY include secret IDs, requirement status, and
  validation failure categories, and MUST NOT include secret values, source
  paths, staged paths, value hashes, provider config contents, or derived boot
  arguments containing secret material

#### Scenario: Provider receives a secret
- **WHEN** a provider requires file delivery for a validated secret
- **THEN** bootup SHALL provide the provider a private handle or staged private
  file path and SHALL NOT place the secret value in public boot plan fields
