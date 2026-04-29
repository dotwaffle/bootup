# VM tests

VM tests are behind the `vmtest` build tag because they require QEMU and a
kernel. The default unit test suite does not boot a VM.

Build an initramfs that reaches the text interface without touching the host
network:

```sh
BOOTUP_UINITCMD='bootup --mode=list-targets' scripts/build-initramfs.sh
```

Run tagged tests through `runvmtest`:

```sh
go run github.com/hugelgupf/vmtest/tools/runvmtest@latest -- \
  go test -tags vmtest ./test/vmtest
```

The tests expect the VM to reach the serial text interface and list the Debian
trixie amd64 provider target.
