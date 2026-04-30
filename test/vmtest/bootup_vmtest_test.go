//go:build vmtest

package vmtest_test

import (
	"os"
	"testing"

	"github.com/hugelgupf/vmtest/qemu"
)

func killAndWait(t *testing.T, vm *qemu.VM) {
	t.Helper()
	if err := vm.Kill(); err != nil {
		t.Fatalf("kill VM: %v", err)
	}
	_ = vm.Wait()
}

func TestBootupReachesTextInterface(t *testing.T) {
	qemu.SkipWithoutQEMU(t)
	if os.Getenv("VMTEST_INITRAMFS") == "" {
		t.Skip("VMTEST_INITRAMFS is required")
	}

	vm := qemu.StartT(t, "bootup", qemu.ArchUseEnvv,
		qemu.WithAppendKernel("console=ttyS0"),
		qemu.LogSerialByLine(qemu.DefaultPrint("bootup", t.Logf)),
	)
	if _, err := vm.Console.ExpectString("bootup targets"); err != nil {
		killAndWait(t, vm)
		t.Fatalf("expect text interface: %v", err)
	}
	killAndWait(t, vm)
}

func TestBootupListsDefaultCatalogTargets(t *testing.T) {
	qemu.SkipWithoutQEMU(t)
	if os.Getenv("VMTEST_INITRAMFS") == "" {
		t.Skip("VMTEST_INITRAMFS is required")
	}

	vm := qemu.StartT(t, "bootup", qemu.ArchUseEnvv,
		qemu.WithAppendKernel("console=ttyS0"),
		qemu.LogSerialByLine(qemu.DefaultPrint("bootup", t.Logf)),
	)
	if _, err := vm.Console.ExpectString("debian-trixie-amd64-netboot"); err != nil {
		killAndWait(t, vm)
		t.Fatalf("expect Debian target: %v", err)
	}
	if _, err := vm.Console.ExpectString("debian/trixie/amd64/installer"); err != nil {
		killAndWait(t, vm)
		t.Fatalf("expect Debian catalog label: %v", err)
	}
	if _, err := vm.Console.ExpectString("ubuntu-2604-amd64-netboot"); err != nil {
		killAndWait(t, vm)
		t.Fatalf("expect Ubuntu target: %v", err)
	}
	if _, err := vm.Console.ExpectString("ubuntu/26.04/amd64/installer"); err != nil {
		killAndWait(t, vm)
		t.Fatalf("expect Ubuntu catalog label: %v", err)
	}
	killAndWait(t, vm)
}

func TestBootupStagesDebianFixture(t *testing.T) {
	qemu.SkipWithoutQEMU(t)
	initramfs := os.Getenv("VMTEST_STAGE_INITRAMFS")
	if initramfs == "" {
		t.Skip("VMTEST_STAGE_INITRAMFS is required")
	}
	t.Setenv("VMTEST_INITRAMFS", initramfs)

	vm := qemu.StartT(t, "bootup", qemu.ArchUseEnvv,
		qemu.WithAppendKernel("console=ttyS0"),
		qemu.LogSerialByLine(qemu.DefaultPrint("bootup", t.Logf)),
	)
	if _, err := vm.Console.ExpectString("kernel\t/tmp/bootup/linux"); err != nil {
		killAndWait(t, vm)
		t.Fatalf("expect staged kernel: %v", err)
	}
	if _, err := vm.Console.ExpectString("initrd\t/tmp/bootup/initrd.gz"); err != nil {
		killAndWait(t, vm)
		t.Fatalf("expect staged initrd: %v", err)
	}
	killAndWait(t, vm)
}

func TestBootupAttemptsRealDebianKexec(t *testing.T) {
	qemu.SkipWithoutQEMU(t)
	initramfs := os.Getenv("VMTEST_REAL_DEBIAN_INITRAMFS")
	if initramfs == "" {
		t.Skip("VMTEST_REAL_DEBIAN_INITRAMFS is required")
	}
	t.Setenv("VMTEST_INITRAMFS", initramfs)

	vm := qemu.StartT(t, "bootup", qemu.ArchUseEnvv,
		qemu.WithAppendKernel("console=ttyS0 panic=30"),
		qemu.WithQEMUArgs("-netdev", "user,id=net0", "-device", "e1000,netdev=net0"),
		qemu.LogSerialByLine(qemu.DefaultPrint("bootup", t.Logf)),
	)
	if _, err := vm.Console.ExpectString("[loading] Debian trixie amd64 netboot"); err != nil {
		killAndWait(t, vm)
		t.Fatalf("expect kexec loading status: %v", err)
	}
	killAndWait(t, vm)
}
