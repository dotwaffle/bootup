// Package handoff transfers control to verified boot targets.
package handoff

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"unsafe"

	"github.com/dotwaffle/bootup/internal/provider"
	urootboot "github.com/u-root/u-root/pkg/boot"
	"golang.org/x/sys/unix"
)

// Loader loads and executes a kexec image.
type Loader interface {
	Load(kernel *os.File, initrd *os.File, cmdline string) error
	Execute() error
}

// KexecExecutor executes a staged boot plan through in-process kexec syscalls.
type KexecExecutor struct {
	Loader            Loader
	LoadSyscallLoader Loader
	LocalBooter       LocalBooter
}

// Execute loads the plan into kexec and executes it.
func (e KexecExecutor) Execute(ctx context.Context, plan provider.BootPlan) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	switch plan.ResolvedAction() {
	case provider.BootActionLinuxKexec:
		return e.executeLinuxKexec(plan)
	case provider.BootActionLocalBoot:
		return e.executeLocalBoot(ctx, plan)
	default:
		return fmt.Errorf("unsupported boot action %q", plan.Action)
	}
}

func (e KexecExecutor) executeLinuxKexec(plan provider.BootPlan) error {
	kernel, err := os.Open(plan.Kernel.Path)
	if err != nil {
		return fmt.Errorf("open kernel: %w", err)
	}
	defer func() { _ = kernel.Close() }()

	var initrd *os.File
	if plan.Initrd.Path != "" {
		initrd, err = os.Open(plan.Initrd.Path)
		if err != nil {
			return fmt.Errorf("open initrd: %w", err)
		}
		defer func() { _ = initrd.Close() }()
	}

	loader := e.Loader
	if loader == nil {
		loader = LinuxKexecFileLoader{}
	}
	if err := loader.Load(kernel, initrd, plan.Cmdline); err != nil {
		if !errors.Is(err, unix.ENOEXEC) {
			return fmt.Errorf("load kexec image: %w", err)
		}
		if err := rewindBootFiles(kernel, initrd); err != nil {
			return fmt.Errorf("prepare kexec_load fallback: %w", err)
		}
		fallbackLoader := e.LoadSyscallLoader
		if fallbackLoader == nil {
			fallbackLoader = LinuxKexecLoadLoader{}
		}
		if fallbackErr := fallbackLoader.Load(kernel, initrd, plan.Cmdline); fallbackErr != nil {
			return fmt.Errorf("load kexec image with kexec_load fallback: %w", errors.Join(
				fmt.Errorf("kexec_file_load: %w", err),
				fmt.Errorf("kexec_load: %w", fallbackErr),
			))
		}
		if fallbackErr := fallbackLoader.Execute(); fallbackErr != nil {
			return fmt.Errorf("execute kexec image with kexec_load fallback: %w", fallbackErr)
		}
		return nil
	}
	if err := loader.Execute(); err != nil {
		return fmt.Errorf("execute kexec image: %w", err)
	}
	return nil
}

func rewindBootFiles(files ...*os.File) error {
	for _, file := range files {
		if file == nil {
			continue
		}
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return fmt.Errorf("seek %s: %w", file.Name(), err)
		}
	}
	return nil
}

// LinuxKexecLoadLoader uses u-root's kexec_load Linux loader.
type LinuxKexecLoadLoader struct{}

// Load loads the Linux kernel and initrd using kexec_load.
func (LinuxKexecLoadLoader) Load(kernel *os.File, initrd *os.File, cmdline string) error {
	image := &urootboot.LinuxImage{
		Kernel:      kernel,
		Initrd:      initrd,
		Cmdline:     cmdline,
		LoadSyscall: true,
	}
	return image.Load()
}

// Execute enters the previously loaded kexec image.
func (LinuxKexecLoadLoader) Execute() error {
	return urootboot.Execute()
}

func (e KexecExecutor) executeLocalBoot(ctx context.Context, plan provider.BootPlan) error {
	localBooter := e.LocalBooter
	if localBooter == nil {
		localBooter = CommandLocalBooter{}
	}
	if err := localBooter.Boot(ctx, plan.Cmdline); err != nil {
		return fmt.Errorf("execute local boot: %w", err)
	}
	return nil
}

// LocalBooter boots from local storage.
type LocalBooter interface {
	Boot(context.Context, string) error
}

// CommandLocalBooter invokes a local boot command from the initramfs.
type CommandLocalBooter struct {
	Command string
	Args    []string
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
}

// Boot executes the configured local boot command.
func (b CommandLocalBooter) Boot(ctx context.Context, cmdline string) error {
	command := b.Command
	if command == "" {
		command = "boot"
	}
	args := append([]string(nil), b.Args...)
	if strings.TrimSpace(cmdline) != "" {
		args = append(args, "-append", cmdline)
	}
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Stdin = b.Stdin
	if cmd.Stdin == nil {
		cmd.Stdin = os.Stdin
	}
	cmd.Stdout = b.Stdout
	if cmd.Stdout == nil {
		cmd.Stdout = os.Stdout
	}
	cmd.Stderr = b.Stderr
	if cmd.Stderr == nil {
		cmd.Stderr = os.Stderr
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %v: %w", command, args, err)
	}
	return nil
}

// LinuxKexecFileLoader uses kexec_file_load and reboot(KEXEC).
type LinuxKexecFileLoader struct{}

// Load loads the kernel and initrd using kexec_file_load.
func (LinuxKexecFileLoader) Load(kernel *os.File, initrd *os.File, cmdline string) error {
	cmdlineBytes := append([]byte(cmdline), 0)
	initrdFD := ^uintptr(0)
	if initrd != nil {
		initrdFD = initrd.Fd()
	}
	_, _, errno := unix.Syscall6(
		unix.SYS_KEXEC_FILE_LOAD,
		kernel.Fd(),
		initrdFD,
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
