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

This keeps handoff inside the bootup binary. If all load attempts fail, or if
the final reboot fails, the executor returns the error without clearing the
current process state.
