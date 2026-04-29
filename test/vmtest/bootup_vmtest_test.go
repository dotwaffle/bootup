//go:build vmtest

package vmtest_test

import (
	"os"
	"testing"

	"github.com/hugelgupf/vmtest/qemu"
)

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
		vm.Kill()
		t.Fatalf("expect text interface: %v", err)
	}
	if err := vm.Kill(); err != nil {
		t.Fatalf("kill VM: %v", err)
	}
}

func TestBootupListsDebianProvider(t *testing.T) {
	qemu.SkipWithoutQEMU(t)
	if os.Getenv("VMTEST_INITRAMFS") == "" {
		t.Skip("VMTEST_INITRAMFS is required")
	}

	vm := qemu.StartT(t, "bootup", qemu.ArchUseEnvv,
		qemu.WithAppendKernel("console=ttyS0"),
		qemu.LogSerialByLine(qemu.DefaultPrint("bootup", t.Logf)),
	)
	if _, err := vm.Console.ExpectString("debian-trixie-amd64-netboot"); err != nil {
		vm.Kill()
		t.Fatalf("expect Debian target: %v", err)
	}
	if err := vm.Kill(); err != nil {
		t.Fatalf("kill VM: %v", err)
	}
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
		vm.Kill()
		t.Fatalf("expect staged kernel: %v", err)
	}
	if _, err := vm.Console.ExpectString("initrd\t/tmp/bootup/initrd.gz"); err != nil {
		vm.Kill()
		t.Fatalf("expect staged initrd: %v", err)
	}
	if err := vm.Kill(); err != nil {
		t.Fatalf("kill VM: %v", err)
	}
}
