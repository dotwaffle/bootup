# Kexec handoff

The handoff path uses in-process Linux syscalls behind
`internal/handoff.KexecExecutor`. Bootup does not package or execute an
external `kexec` binary.

Bootup stages provider-verified kernel and initrd files on disk, then first
calls:

```text
kexec_file_load(kernel_fd, initrd_fd, cmdline)
reboot(LINUX_REBOOT_CMD_KEXEC)
```

If `kexec_file_load` rejects the image format with `ENOEXEC`, bootup retries
the load through u-root's Linux `kexec_load` path before rebooting. Other load
errors, such as Secure Boot lockdown, missing capability, or missing kernel
support, fail without retrying because the fallback is not expected to change
policy or permission failures.

The `kexec_load` fallback is still a Linux kernel handoff, not a firmware,
memdisk, COM32, or generic bootloader emulator. u-root's Multiboot package is
useful for Multiboot v1 kernels with modules, but it does not make stock
FreeBSD release artifacts bootable because they depend on FreeBSD loader
semantics. For another example, MemTest86+ 8.00's x86_64 image boots through
QEMU's `-kernel` Linux boot protocol path, but it is not accepted by
`kexec_file_load` because it does not set `XLF_CAN_BE_LOADED_ABOVE_4G`, and
u-root's Linux `kexec_load` path does not support the image's non-relocatable
form. That class of payload needs a separate boot action before it can be
advertised as a bootup target.

This keeps handoff inside the bootup binary. If all load attempts fail, or if
the final reboot fails, the executor returns the error without clearing the
current process state.
