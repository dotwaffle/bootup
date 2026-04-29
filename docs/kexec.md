# Kexec handoff

The MVP handoff path uses in-process Linux syscalls behind
`internal/handoff.KexecExecutor`. Bootup does not package or execute an
external `kexec` binary.

Bootup stages provider-verified kernel and initrd files on disk, then calls:

```text
kexec_file_load(kernel_fd, initrd_fd, cmdline)
reboot(LINUX_REBOOT_CMD_KEXEC)
```

This keeps handoff inside the bootup binary. If Secure Boot, kernel lockdown,
or missing kernel support prevents `kexec`, the executor returns the load or
execute error without clearing the current process state.

Future work can replace the syscall implementation with u-root boot package
integration without changing the provider boot plan contract.
