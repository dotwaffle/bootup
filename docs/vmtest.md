# VM tests

VM tests are behind the `vmtest` build tag because they require QEMU and a
kernel. The default unit test suite does not boot a VM.

Build an initramfs that reaches the text interface without touching the host
network:

```sh
scripts/build-initramfs.sh dist/bootup-initramfs.cpio 'bootup --mode=list-targets' ''
```

Run tagged tests through `runvmtest`:

```sh
go run github.com/hugelgupf/vmtest/tools/runvmtest@latest -- \
  go test -tags vmtest ./test/vmtest
```

The tests expect the VM to reach the serial text interface and list the Debian
trixie amd64 provider target.

Build a second hermetic fixture initramfs that selects Debian and stages signed
fixture artifacts through the real Debian provider code:

```sh
scripts/build-initramfs.sh dist/bootup-fixture-initramfs.cpio 'bootup --mode=stage-target --target=debian-trixie-amd64-netboot --staging-dir=/tmp/bootup' bootup_debian_fixture
```

Then run vmtest with both initramfs paths:

```sh
VMTEST_QEMU=qemu-system-x86_64 \
VMTEST_INITRAMFS=dist/bootup-initramfs.cpio.zst \
VMTEST_STAGE_INITRAMFS=dist/bootup-fixture-initramfs.cpio.zst \
go run github.com/hugelgupf/vmtest/tools/runvmtest@latest -- \
  go test -tags vmtest ./test/vmtest
```
