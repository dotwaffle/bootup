## MODIFIED Requirements

### Requirement: Network and time preparation
Bootup SHALL prepare enough network and time state to perform trusted remote
artifact retrieval.

#### Scenario: Network is already configured
- **WHEN** bootup starts with kernel or loader-provided network configuration
- **THEN** bootup SHALL validate that a non-loopback network interface is
  configured before provider discovery or artifact retrieval

#### Scenario: Kernel DNS hints are available
- **WHEN** the kernel provides DNS hints through `/proc/net/pnp`
- **THEN** bootup SHALL make those hints available to normal resolver lookup
  when no resolver configuration exists

#### Scenario: Time is not trustworthy
- **WHEN** bootup detects that system time is unsuitable for TLS or signature
  validation workflows
- **THEN** bootup SHALL attempt a configured time synchronization path before
  continuing network artifact retrieval
