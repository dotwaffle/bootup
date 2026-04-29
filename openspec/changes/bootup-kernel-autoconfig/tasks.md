## 1. Kernel Configuration Assets

- [x] 1.1 Add a bootup kernel config fragment for built-in IPv4 DHCP autoconfig
- [x] 1.2 Include built-in QEMU-friendly NIC driver requirements
- [x] 1.3 Ensure no compiled kernel or module artifacts are added

## 2. Local Validation

- [x] 2.1 Add a script that checks a kernel config file for required built-in symbols
- [x] 2.2 Add fixture tests for passing, missing, and modular kernel config cases
- [x] 2.3 Document the validation command

## 3. Launch Documentation

- [x] 3.1 Document `ip=::::::dhcp` as the purpose-built kernel default
- [x] 3.2 Keep the host-kernel QEMU static-network fallback documented
- [x] 3.3 Update examples to include `panic=30` consistently

## 4. Verification

- [x] 4.1 Run shell syntax checks for all scripts
- [x] 4.2 Run the default Go tests and linters
- [x] 4.3 Mark OpenSpec tasks complete after validation
