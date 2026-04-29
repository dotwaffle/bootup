// Package handoff transfers control to verified boot targets.
package handoff

import (
	"context"
	"fmt"
	"os"
	"unsafe"

	"github.com/dotwaffle/bootup/internal/provider"
	"golang.org/x/sys/unix"
)

// Loader loads and executes a kexec image.
type Loader interface {
	Load(kernel *os.File, initrd *os.File, cmdline string) error
	Execute() error
}

// KexecExecutor executes a staged boot plan through in-process kexec syscalls.
type KexecExecutor struct {
	Loader Loader
}

// Execute loads the plan into kexec and executes it.
func (e KexecExecutor) Execute(ctx context.Context, plan provider.BootPlan) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	kernel, err := os.Open(plan.Kernel.Path)
	if err != nil {
		return fmt.Errorf("open kernel: %w", err)
	}
	defer func() { _ = kernel.Close() }()

	initrd, err := os.Open(plan.Initrd.Path)
	if err != nil {
		return fmt.Errorf("open initrd: %w", err)
	}
	defer func() { _ = initrd.Close() }()

	loader := e.Loader
	if loader == nil {
		loader = LinuxKexecFileLoader{}
	}
	if err := loader.Load(kernel, initrd, plan.Cmdline); err != nil {
		return fmt.Errorf("load kexec image: %w", err)
	}
	if err := loader.Execute(); err != nil {
		return fmt.Errorf("execute kexec image: %w", err)
	}
	return nil
}

// LinuxKexecFileLoader uses kexec_file_load and reboot(KEXEC).
type LinuxKexecFileLoader struct{}

// Load loads the kernel and initrd using kexec_file_load.
func (LinuxKexecFileLoader) Load(kernel *os.File, initrd *os.File, cmdline string) error {
	cmdlineBytes := append([]byte(cmdline), 0)
	_, _, errno := unix.Syscall6(
		unix.SYS_KEXEC_FILE_LOAD,
		kernel.Fd(),
		initrd.Fd(),
		uintptr(len(cmdlineBytes)),
		uintptr(unsafe.Pointer(&cmdlineBytes[0])),
		0,
		0,
	)
	if errno != 0 {
		return errno
	}
	return nil
}

// Execute enters the previously loaded kexec image.
func (LinuxKexecFileLoader) Execute() error {
	return unix.Reboot(unix.LINUX_REBOOT_CMD_KEXEC)
}
