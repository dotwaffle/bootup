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

The tests expect the VM to reach the serial text interface and list the default
static catalog, including Debian, Fedora, and Ubuntu provider targets.

The repository wrapper builds a default initramfs when needed, queries
kernel.org for the latest stable Linux release, builds and caches the matching
bootup kernel, and runs the tagged VM tests:

```sh
test/vmtest/run
```

Use `BOOTUP_KERNEL_VERSION` to pin a specific upstream kernel version, or
`BOOTUP_VMTEST_CACHE` to change the cache directory.

Build a purpose-built bootup kernel for vmtest/QEMU with Docker:

```sh
scripts/build-kernel.sh
```

Use its output as `VMTEST_KERNEL` or `BOOTUP_KERNEL`:

```sh
VMTEST_KERNEL="$(ls -1 dist/kernel/linux-*-bootup-amd64-bzImage | tail -n 1)" \
go run github.com/hugelgupf/vmtest/tools/runvmtest@latest -- \
  go test -tags vmtest ./test/vmtest
```

The exact version in the output path follows kernel.org's latest stable release
unless `BOOTUP_KERNEL_VERSION` is set.

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

Compile-only VM tests remain part of normal CI. Actual VM execution is opt-in
because it requires `VMTEST_QEMU`. The Debian fixture initramfs also registers
the Fedora provider against the embedded static catalog, so the fixture build
compiles Fedora provider paths without requiring live Fedora network access.

For live Debian staging outside a VM, enable the opt-in live smoke test:

```sh
BOOTUP_LIVE_DEBIAN_SMOKE=1 go test -count=1 ./test/live
```

For live Ubuntu staging outside a VM, enable the matching opt-in smoke test:

```sh
BOOTUP_LIVE_UBUNTU_SMOKE=1 go test -count=1 ./test/live
```

The Ubuntu live staging test uses the provider's default HTTPS-only netboot
path. It does not require Ubuntu keyring material or pinned artifact hashes.

Catalog target smoke selection is also covered in `./test/live`. The default
test run only verifies that static catalog targets can be selected by target ID
and that unsupported targets are reported without contacting the network. Set
`BOOTUP_LIVE_CATALOG_SMOKE=1` to stage selected generic Linux catalog targets
from upstream mirrors:

```sh
BOOTUP_LIVE_CATALOG_SMOKE=1 go test -count=1 ./test/live
```

That opt-in path currently covers `opensuse-leap-160-amd64-netboot` as a
kernel+initrd target. It requires outbound HTTPS access, working DNS, enough
temporary disk space for downloaded artifacts, and enough time for upstream
mirrors to respond. Keep it out of the default suite because failures can
reflect network or mirror state rather than bootup regressions.

QEMU VM smoke runs additionally require `qemu-system-x86_64`, a host kernel
that can boot as the outer VM kernel, KVM or software emulation capacity, serial
console output, and network configuration inside the VM when the selected target
must fetch artifacts before kexec. Use the explicit environment variables or
helper scripts below so those requirements are visible at invocation time.

To attempt a catalog target through the current QEMU helper by target ID:

```sh
BOOTUP_LIVE_CATALOG_SMOKE=1 scripts/smoke-catalog-target.sh opensuse-leap-160-amd64-netboot
```

For a real Debian kexec VM smoke, first build a Debian-capable initramfs:

```sh
NET_MODULE="$(modinfo -n e1000)"
scripts/build-debian-initramfs.sh \
  /usr/share/keyrings/debian-archive-keyring.gpg \
  dist/bootup-debian-smoke-initramfs.cpio \
  "gosh -c 'insmod ${NET_MODULE} || true; ip link set eth0 up; ip addr add 10.0.2.15/24 dev eth0 || true; ip route add default via 10.0.2.2 dev eth0 || true; echo nameserver 10.0.2.3 >/etc/resolv.conf; bootup --mode=boot-target --target=debian-trixie-amd64-netboot --staging-dir=/tmp/bootup --provider-config=/etc/bootup/providers.json'" \
  "${NET_MODULE}"
```

Then run the opt-in VM test:

```sh
VMTEST_QEMU=qemu-system-x86_64 \
VMTEST_REAL_DEBIAN_INITRAMFS=dist/bootup-debian-smoke-initramfs.cpio.zst \
go run github.com/hugelgupf/vmtest/tools/runvmtest@latest -- \
  go test -count=1 -tags vmtest ./test/vmtest
```

For a real Ubuntu kexec VM smoke, use the helper:

```sh
scripts/smoke-real-ubuntu.sh
```

The helper exits with the timeout status if QEMU remains in the target
installer after a successful kexec. Check the serial output for
`[loading] Ubuntu 26.04 amd64 netboot` and the target kernel boot log.
