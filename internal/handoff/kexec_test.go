package handoff_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/dotwaffle/bootup/internal/handoff"
	"github.com/dotwaffle/bootup/internal/provider"
	"golang.org/x/sys/unix"
)

type fakeLoader struct {
	loads    []loadCall
	executes int
	err      error
}

type fakeLocalBooter struct {
	cmdlines []string
	err      error
}

type loadCall struct {
	kernel  string
	initrd  string
	cmdline string
}

func (l *fakeLoader) Load(kernel *os.File, initrd *os.File, cmdline string) error {
	call := loadCall{kernel: kernel.Name(), cmdline: cmdline}
	if initrd != nil {
		call.initrd = initrd.Name()
	}
	l.loads = append(l.loads, call)
	return l.err
}

func (l *fakeLoader) Execute() error {
	l.executes++
	return l.err
}

func (b *fakeLocalBooter) Boot(_ context.Context, cmdline string) error {
	b.cmdlines = append(b.cmdlines, cmdline)
	return b.err
}

func TestKexecLoadsThenExecutesPlan(t *testing.T) {
	t.Parallel()

	kernel := writeTempFile(t, "linux")
	initrd := writeTempFile(t, "initrd.gz")
	loader := &fakeLoader{}
	executor := handoff.KexecExecutor{Loader: loader}
	plan := provider.BootPlan{
		Kernel:  provider.Artifact{Path: kernel},
		Initrd:  provider.Artifact{Path: initrd},
		Cmdline: "priority=low",
	}

	if err := executor.Execute(context.Background(), plan); err != nil {
		t.Fatalf("execute kexec: %v", err)
	}

	if len(loader.loads) != 1 {
		t.Fatalf("loads = %d, want 1", len(loader.loads))
	}
	if loader.loads[0].kernel != kernel {
		t.Fatalf("kernel = %q, want %q", loader.loads[0].kernel, kernel)
	}
	if loader.loads[0].cmdline != "priority=low" {
		t.Fatalf("cmdline = %q, want priority=low", loader.loads[0].cmdline)
	}
	if loader.executes != 1 {
		t.Fatalf("executes = %d, want 1", loader.executes)
	}
}

func TestKexecAllowsKernelOnlyPlan(t *testing.T) {
	t.Parallel()

	kernel := writeTempFile(t, "mt86plus")
	loader := &fakeLoader{}
	executor := handoff.KexecExecutor{Loader: loader}
	plan := provider.BootPlan{
		Action:  provider.BootActionLinuxKexec,
		Kernel:  provider.Artifact{Path: kernel},
		Cmdline: "console=ttyS0",
	}

	if err := executor.Execute(context.Background(), plan); err != nil {
		t.Fatalf("execute kexec: %v", err)
	}

	if len(loader.loads) != 1 {
		t.Fatalf("loads = %d, want 1", len(loader.loads))
	}
	if loader.loads[0].initrd != "" {
		t.Fatalf("initrd = %q, want none", loader.loads[0].initrd)
	}
	if loader.executes != 1 {
		t.Fatalf("executes = %d, want 1", loader.executes)
	}
}

func TestKexecReportsLoadFailure(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("lockdown")
	executor := handoff.KexecExecutor{Loader: &fakeLoader{err: wantErr}}

	err := executor.Execute(context.Background(), provider.BootPlan{
		Kernel: provider.Artifact{Path: writeTempFile(t, "linux")},
		Initrd: provider.Artifact{Path: writeTempFile(t, "initrd.gz")},
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("execute error = %v, want wrapped %v", err, wantErr)
	}
}

func TestKexecFallsBackToLoadSyscallWhenFileLoadRejectsImage(t *testing.T) {
	t.Parallel()

	kernel := writeTempFile(t, "mt86plus")
	fileLoader := &fakeLoader{err: unix.ENOEXEC}
	loadSyscallLoader := &fakeLoader{}
	executor := handoff.KexecExecutor{
		Loader:            fileLoader,
		LoadSyscallLoader: loadSyscallLoader,
	}

	err := executor.Execute(context.Background(), provider.BootPlan{
		Action:  provider.BootActionLinuxKexec,
		Kernel:  provider.Artifact{Path: kernel},
		Cmdline: "console=ttyS0",
	})
	if err != nil {
		t.Fatalf("execute kexec fallback: %v", err)
	}

	if len(fileLoader.loads) != 1 {
		t.Fatalf("file loader loads = %d, want 1", len(fileLoader.loads))
	}
	if fileLoader.executes != 0 {
		t.Fatalf("file loader executes = %d, want 0", fileLoader.executes)
	}
	if len(loadSyscallLoader.loads) != 1 {
		t.Fatalf("load syscall loader loads = %d, want 1", len(loadSyscallLoader.loads))
	}
	if loadSyscallLoader.loads[0].kernel != kernel {
		t.Fatalf("fallback kernel = %q, want %q", loadSyscallLoader.loads[0].kernel, kernel)
	}
	if loadSyscallLoader.loads[0].cmdline != "console=ttyS0" {
		t.Fatalf("fallback cmdline = %q, want console=ttyS0", loadSyscallLoader.loads[0].cmdline)
	}
	if loadSyscallLoader.executes != 1 {
		t.Fatalf("load syscall loader executes = %d, want 1", loadSyscallLoader.executes)
	}
}

func TestKexecDoesNotFallbackForNonFormatLoadFailure(t *testing.T) {
	t.Parallel()

	fileLoader := &fakeLoader{err: unix.EPERM}
	loadSyscallLoader := &fakeLoader{}
	executor := handoff.KexecExecutor{
		Loader:            fileLoader,
		LoadSyscallLoader: loadSyscallLoader,
	}

	err := executor.Execute(context.Background(), provider.BootPlan{
		Kernel: provider.Artifact{Path: writeTempFile(t, "linux")},
	})
	if !errors.Is(err, unix.EPERM) {
		t.Fatalf("execute error = %v, want wrapped EPERM", err)
	}
	if len(loadSyscallLoader.loads) != 0 || loadSyscallLoader.executes != 0 {
		t.Fatalf("fallback used after EPERM: loads=%#v executes=%d", loadSyscallLoader.loads, loadSyscallLoader.executes)
	}
}

func TestKexecReportsFallbackLoadFailure(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("fallback refused image")
	executor := handoff.KexecExecutor{
		Loader:            &fakeLoader{err: unix.ENOEXEC},
		LoadSyscallLoader: &fakeLoader{err: wantErr},
	}

	err := executor.Execute(context.Background(), provider.BootPlan{
		Kernel: provider.Artifact{Path: writeTempFile(t, "mt86plus")},
	})
	if !errors.Is(err, unix.ENOEXEC) {
		t.Fatalf("execute error = %v, want original ENOEXEC", err)
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("execute error = %v, want wrapped fallback error %v", err, wantErr)
	}
}

func TestExecutorDispatchesLocalBootAction(t *testing.T) {
	t.Parallel()

	loader := &fakeLoader{}
	localBooter := &fakeLocalBooter{}
	executor := handoff.KexecExecutor{
		Loader:      loader,
		LocalBooter: localBooter,
	}

	err := executor.Execute(context.Background(), provider.BootPlan{
		Action:  provider.BootActionLocalBoot,
		Cmdline: "console=ttyS0",
	})
	if err != nil {
		t.Fatalf("execute localboot: %v", err)
	}
	if len(localBooter.cmdlines) != 1 || localBooter.cmdlines[0] != "console=ttyS0" {
		t.Fatalf("local boot cmdlines = %#v, want console append", localBooter.cmdlines)
	}
	if len(loader.loads) != 0 || loader.executes != 0 {
		t.Fatalf("kexec loader used for localboot: loads=%#v executes=%d", loader.loads, loader.executes)
	}
}

func TestExecutorRejectsUnsupportedAction(t *testing.T) {
	t.Parallel()

	executor := handoff.KexecExecutor{Loader: &fakeLoader{}}
	err := executor.Execute(context.Background(), provider.BootPlan{
		Action: provider.BootAction("memdisk"),
	})
	if err == nil {
		t.Fatal("execute unsupported action succeeded")
	}
}

func writeTempFile(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(name), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}
