## ADDED Requirements

### Requirement: Action-dispatched handoff
Bootup SHALL dispatch target handoff behavior using the boot action in the
staged boot plan.

#### Scenario: Linux kexec plan is ready
- **WHEN** a staged boot plan has the `linux-kexec` action
- **THEN** bootup SHALL invoke the configured kexec path with the provider's
  kernel, optional initrd, and command line

#### Scenario: Local boot plan is ready
- **WHEN** a staged boot plan has the `localboot` action
- **THEN** bootup SHALL invoke the configured local boot path without requiring
  downloaded artifacts

### Requirement: Optional initrd artifact
Bootup SHALL allow Linux kexec targets to omit an initrd when the selected
kernel can boot without one.

#### Scenario: Kernel-only plan is staged
- **WHEN** a Linux kexec plan has a kernel artifact and no initrd artifact
- **THEN** bootup SHALL stage the kernel and execute kexec with no initrd file

### Requirement: Operator command-line append
Bootup SHALL allow operators to append additional command-line parameters to
the selected boot plan before staging and handoff.

#### Scenario: Additional parameters are configured
- **WHEN** bootup plans a target and operator command-line parameters are set
- **THEN** bootup SHALL append those parameters to the provider command line
  before staging artifacts or executing handoff

### Requirement: Explicit network reconfiguration
Bootup SHALL allow operators to configure interface address, default route, and
DNS before provider discovery or artifact retrieval.

#### Scenario: Network configuration is supplied
- **WHEN** bootup starts with explicit network configuration
- **THEN** it SHALL apply that configuration before validating network
  readiness
