package app_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	expect "github.com/Netflix/go-expect"
	"github.com/creack/pty"
	"github.com/dotwaffle/bootup/internal/app"
	"github.com/dotwaffle/bootup/internal/provider"
)

type providerStub struct {
	id      string
	targets []provider.Target
	plan    provider.BootPlan
	staged  provider.BootPlan
	planned *provider.Target
}

func (p providerStub) ID() string {
	if p.id != "" {
		return p.id
	}
	return "debian"
}

func (p providerStub) Targets(context.Context) ([]provider.Target, error) {
	return p.targets, nil
}

func (p providerStub) Plan(_ context.Context, target provider.Target) (provider.BootPlan, error) {
	if p.planned != nil {
		*p.planned = target
	}
	return p.plan, nil
}

func (p providerStub) Stage(context.Context, provider.StageConfig) (provider.BootPlan, error) {
	return p.staged, nil
}

type discoveryProviderStub struct {
	providerStub
	family      provider.DiscoveryFamily
	discovered  []provider.Target
	discoverErr error
}

func (p discoveryProviderStub) DiscoveryFamily() provider.DiscoveryFamily {
	return p.family
}

func (p discoveryProviderStub) DiscoverTargets(context.Context) ([]provider.Target, error) {
	return p.discovered, p.discoverErr
}

type executorStub struct {
	executed *provider.BootPlan
	err      error
}

func (e executorStub) Execute(_ context.Context, plan provider.BootPlan) error {
	if e.executed != nil {
		*e.executed = plan
	}
	return e.err
}

func debianTarget() provider.Target {
	return provider.Target{
		ID:         "debian-trixie-amd64-netboot",
		ProviderID: "debian",
		Name:       "Debian trixie amd64 netboot",
		Catalog: provider.CatalogEntry{
			Distribution: "debian",
			Release:      "trixie",
			Architecture: "amd64",
			Kind:         "installer",
		},
	}
}

func ubuntuTarget() provider.Target {
	return provider.Target{
		ID:         "ubuntu-2604-amd64-netboot",
		ProviderID: "ubuntu",
		Name:       "Ubuntu 26.04 amd64 netboot",
		Catalog: provider.CatalogEntry{
			Distribution: "ubuntu",
			Release:      "26.04",
			Architecture: "amd64",
			Kind:         "installer",
		},
	}
}

func TestRunListsTargetsInNonInteractiveMode(t *testing.T) {
	t.Parallel()

	registry := provider.NewRegistry()
	if err := registry.Register(providerStub{targets: []provider.Target{debianTarget()}}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := app.New(app.Config{
		Registry: registry,
		Stdout:   &stdout,
		Stderr:   &stderr,
		Logger:   slog.New(slog.NewTextHandler(&stderr, nil)),
		Mode:     app.ModeListTargets,
	})

	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("run app: %v", err)
	}

	if !strings.Contains(stdout.String(), "Debian trixie") {
		t.Fatalf("stdout = %q, want Debian target", stdout.String())
	}
	if !strings.Contains(stdout.String(), "debian-trixie-amd64-netboot") {
		t.Fatalf("stdout = %q, want target id", stdout.String())
	}
}

func TestRunPreparesRuntimeBeforeListingTargets(t *testing.T) {
	t.Parallel()

	var calls []string
	registry := provider.NewRegistry()
	if err := registry.Register(providerStub{targets: []provider.Target{debianTarget()}}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	runner := app.New(app.Config{
		Registry: registry,
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		Logger:   slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
		Mode:     app.ModeListTargets,
		Preparers: []app.Preparer{
			app.PrepareFunc(func(context.Context) error {
				calls = append(calls, "prepare")
				return nil
			}),
		},
		OnBeforeListTargets: func() {
			calls = append(calls, "targets")
		},
	})

	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("run app: %v", err)
	}

	if len(calls) != 2 || calls[0] != "prepare" || calls[1] != "targets" {
		t.Fatalf("calls = %#v, want prepare before targets", calls)
	}
}

func TestRunHoldsAfterModeCompletes(t *testing.T) {
	t.Parallel()

	registry := provider.NewRegistry()
	if err := registry.Register(providerStub{targets: []provider.Target{debianTarget()}}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	var stdout bytes.Buffer
	runner := app.New(app.Config{
		Registry: registry,
		Stdout:   &stdout,
		Stderr:   &bytes.Buffer{},
		Logger:   slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
		Mode:     app.ModeListTargets,
		Hold:     true,
		OnBeforeListTargets: func() {
			cancel()
		},
	})

	err := runner.Run(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("run app error = %v, want context canceled", err)
	}
	if !strings.Contains(stdout.String(), "Debian trixie") {
		t.Fatalf("stdout = %q, want target list before hold", stdout.String())
	}
}

func TestRunPlansSelectedTargetInNonInteractiveMode(t *testing.T) {
	t.Parallel()

	target := debianTarget()
	plan := provider.BootPlan{
		Target:  target,
		Kernel:  provider.Artifact{Name: "linux", URL: "https://example.test/linux"},
		Initrd:  provider.Artifact{Name: "initrd.gz", URL: "https://example.test/initrd.gz"},
		Cmdline: "priority=low",
	}
	var planned provider.Target

	registry := provider.NewRegistry()
	if err := registry.Register(providerStub{
		targets: []provider.Target{target},
		plan:    plan,
		planned: &planned,
	}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	var stdout bytes.Buffer
	runner := app.New(app.Config{
		Registry: registry,
		Stdout:   &stdout,
		Stderr:   &bytes.Buffer{},
		Logger:   slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
		Mode:     app.ModePlanTarget,
		TargetID: "debian-trixie-amd64-netboot",
	})

	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("run app: %v", err)
	}
	if planned.ID != target.ID {
		t.Fatalf("planned target = %#v, want %#v", planned, target)
	}
	if !strings.Contains(stdout.String(), "https://example.test/linux") {
		t.Fatalf("stdout = %q, want kernel URL", stdout.String())
	}
}

func TestRunStagesSelectedTargetInNonInteractiveMode(t *testing.T) {
	t.Parallel()

	target := debianTarget()
	plan := provider.BootPlan{Target: target}
	staged := provider.BootPlan{
		Target:  target,
		Kernel:  provider.Artifact{Name: "linux", Path: "/tmp/bootup/linux"},
		Initrd:  provider.Artifact{Name: "initrd.gz", Path: "/tmp/bootup/initrd.gz"},
		Cmdline: "priority=low",
	}

	registry := provider.NewRegistry()
	if err := registry.Register(providerStub{
		targets: []provider.Target{target},
		plan:    plan,
		staged:  staged,
	}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	var stdout bytes.Buffer
	runner := app.New(app.Config{
		Registry:   registry,
		Stdout:     &stdout,
		Stderr:     &bytes.Buffer{},
		Logger:     slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
		Mode:       app.ModeStageTarget,
		TargetID:   "debian-trixie-amd64-netboot",
		StagingDir: t.TempDir(),
	})

	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("run app: %v", err)
	}
	if !strings.Contains(stdout.String(), "/tmp/bootup/linux") {
		t.Fatalf("stdout = %q, want staged kernel path", stdout.String())
	}
	for _, phase := range []string{"[planning]", "[verifying]", "[staging]"} {
		if !strings.Contains(stdout.String(), phase) {
			t.Fatalf("stdout = %q, want phase %s", stdout.String(), phase)
		}
	}
}

func TestRunBootsSelectedTargetInNonInteractiveMode(t *testing.T) {
	t.Parallel()

	target := debianTarget()
	staged := provider.BootPlan{
		Target:  target,
		Kernel:  provider.Artifact{Name: "linux", Path: "/tmp/bootup/linux"},
		Initrd:  provider.Artifact{Name: "initrd.gz", Path: "/tmp/bootup/initrd.gz"},
		Cmdline: "priority=low",
	}
	var executed provider.BootPlan

	registry := provider.NewRegistry()
	if err := registry.Register(providerStub{
		targets: []provider.Target{target},
		plan:    provider.BootPlan{Target: target},
		staged:  staged,
	}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	runner := app.New(app.Config{
		Registry:   registry,
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		Logger:     slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
		Mode:       app.ModeBootTarget,
		TargetID:   "debian-trixie-amd64-netboot",
		StagingDir: t.TempDir(),
		Executor:   executorStub{executed: &executed},
	})

	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("run app: %v", err)
	}
	if executed.Kernel.Path != staged.Kernel.Path {
		t.Fatalf("executed plan = %#v, want %#v", executed, staged)
	}
}

func TestRunMenuSelectsAndBootsTarget(t *testing.T) {
	t.Parallel()

	target := debianTarget()
	staged := provider.BootPlan{
		Target:  target,
		Kernel:  provider.Artifact{Name: "linux", Path: "/tmp/bootup/linux"},
		Initrd:  provider.Artifact{Name: "initrd.gz", Path: "/tmp/bootup/initrd.gz"},
		Cmdline: "priority=low",
	}
	var executed provider.BootPlan

	registry := provider.NewRegistry()
	if err := registry.Register(providerStub{
		targets: []provider.Target{target},
		plan:    provider.BootPlan{Target: target},
		staged:  staged,
	}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	var stdout bytes.Buffer
	runner := app.New(app.Config{
		Registry:   registry,
		Stdin:      strings.NewReader("1\n"),
		Stdout:     &stdout,
		Stderr:     &bytes.Buffer{},
		Logger:     slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
		Mode:       app.ModeMenu,
		StagingDir: t.TempDir(),
		Executor:   executorStub{executed: &executed},
	})

	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("run app: %v", err)
	}
	if executed.Kernel.Path != staged.Kernel.Path {
		t.Fatalf("executed plan = %#v, want %#v", executed, staged)
	}
	if !strings.Contains(stdout.String(), "target> ") {
		t.Fatalf("stdout = %q, want prompt", stdout.String())
	}
}

func TestRunMenuDiscoversFamilyAndBootsTarget(t *testing.T) {
	t.Parallel()

	staticTarget := debianTarget()
	discoveredTarget := provider.Target{
		ID:         "debian-forky-amd64-netboot",
		ProviderID: "debian",
		Name:       "Debian forky amd64 netboot",
		Catalog: provider.CatalogEntry{
			Distribution: "debian",
			Release:      "forky",
			Architecture: "amd64",
			Kind:         "installer",
		},
	}
	staged := provider.BootPlan{
		Target:  discoveredTarget,
		Kernel:  provider.Artifact{Name: "linux", Path: "/tmp/bootup/linux"},
		Initrd:  provider.Artifact{Name: "initrd.gz", Path: "/tmp/bootup/initrd.gz"},
		Cmdline: "priority=low",
	}
	var executed provider.BootPlan

	registry := provider.NewRegistry()
	if err := registry.Register(discoveryProviderStub{
		providerStub: providerStub{
			targets: []provider.Target{staticTarget},
			plan:    provider.BootPlan{Target: discoveredTarget},
			staged:  staged,
		},
		family: provider.DiscoveryFamily{
			ID:         "debian",
			ProviderID: "debian",
			Name:       "Debian",
		},
		discovered: []provider.Target{discoveredTarget},
	}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	var stdout bytes.Buffer
	runner := app.New(app.Config{
		Registry:   registry,
		Stdin:      strings.NewReader("2\n1\n"),
		Stdout:     &stdout,
		Stderr:     &bytes.Buffer{},
		Logger:     slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
		Mode:       app.ModeMenu,
		StagingDir: t.TempDir(),
		Executor:   executorStub{executed: &executed},
	})

	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("run app: %v", err)
	}
	if executed.Target.ID != discoveredTarget.ID {
		t.Fatalf("executed target = %q, want discovered target", executed.Target.ID)
	}
	got := stdout.String()
	for _, want := range []string{"[discovering] Debian", "debian-forky-amd64-netboot"} {
		if !strings.Contains(got, want) {
			t.Fatalf("stdout = %q, want %q", got, want)
		}
	}
}

func TestRunMenuReportsEmptyDiscovery(t *testing.T) {
	t.Parallel()

	registry := provider.NewRegistry()
	if err := registry.Register(discoveryProviderStub{
		providerStub: providerStub{targets: []provider.Target{debianTarget()}},
		family: provider.DiscoveryFamily{
			ID:         "debian",
			ProviderID: "debian",
			Name:       "Debian",
		},
	}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	var stdout bytes.Buffer
	runner := app.New(app.Config{
		Registry: registry,
		Stdin:    strings.NewReader("2\n"),
		Stdout:   &stdout,
		Stderr:   &bytes.Buffer{},
		Logger:   slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
		Mode:     app.ModeMenu,
		UIMode:   app.UIModePlain,
	})

	err := runner.Run(context.Background())
	if err == nil {
		t.Fatal("run app succeeded, want empty discovery error")
	}
	if !strings.Contains(stdout.String(), "no targets discovered for debian") {
		t.Fatalf("stdout = %q, want empty discovery message", stdout.String())
	}
}

func TestRunDiscoversTargetsInNonInteractiveMode(t *testing.T) {
	t.Parallel()

	target := provider.Target{
		ID:         "debian-forky-amd64-netboot",
		ProviderID: "debian",
		Name:       "Debian forky amd64 netboot",
		Catalog: provider.CatalogEntry{
			Distribution: "debian",
			Release:      "forky",
			Architecture: "amd64",
			Kind:         "installer",
		},
	}
	registry := provider.NewRegistry()
	if err := registry.Register(discoveryProviderStub{
		providerStub: providerStub{targets: []provider.Target{debianTarget()}},
		family: provider.DiscoveryFamily{
			ID:         "debian",
			ProviderID: "debian",
			Name:       "Debian",
		},
		discovered: []provider.Target{target},
	}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	var stdout bytes.Buffer
	runner := app.New(app.Config{
		Registry:          registry,
		Stdout:            &stdout,
		Stderr:            &bytes.Buffer{},
		Logger:            slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
		Mode:              app.ModeDiscoverTargets,
		DiscoveryFamilyID: "debian",
	})

	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("run app: %v", err)
	}
	if !strings.Contains(stdout.String(), "debian-forky-amd64-netboot") {
		t.Fatalf("stdout = %q, want discovered target", stdout.String())
	}
}

func TestRunReportsEmptyDiscoveryInNonInteractiveMode(t *testing.T) {
	t.Parallel()

	registry := provider.NewRegistry()
	if err := registry.Register(discoveryProviderStub{
		providerStub: providerStub{targets: []provider.Target{debianTarget()}},
		family: provider.DiscoveryFamily{
			ID:         "debian",
			ProviderID: "debian",
			Name:       "Debian",
		},
	}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	var stdout bytes.Buffer
	runner := app.New(app.Config{
		Registry:          registry,
		Stdout:            &stdout,
		Stderr:            &bytes.Buffer{},
		Logger:            slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
		Mode:              app.ModeDiscoverTargets,
		DiscoveryFamilyID: "debian",
	})

	err := runner.Run(context.Background())
	if err == nil {
		t.Fatal("run app succeeded, want empty discovery error")
	}
	if !strings.Contains(stdout.String(), "no discovered targets for debian") {
		t.Fatalf("stdout = %q, want empty discovery message", stdout.String())
	}
}

func TestRunListsStaticTargetsWhenDiscoveryWouldFail(t *testing.T) {
	t.Parallel()

	registry := provider.NewRegistry()
	if err := registry.Register(discoveryProviderStub{
		providerStub: providerStub{targets: []provider.Target{debianTarget()}},
		family: provider.DiscoveryFamily{
			ID:         "debian",
			ProviderID: "debian",
			Name:       "Debian",
		},
		discoverErr: errors.New("metadata unavailable"),
	}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	var stdout bytes.Buffer
	runner := app.New(app.Config{
		Registry: registry,
		Stdout:   &stdout,
		Stderr:   &bytes.Buffer{},
		Logger:   slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
		Mode:     app.ModeListTargets,
	})

	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("run app: %v", err)
	}
	if !strings.Contains(stdout.String(), "debian-trixie-amd64-netboot") {
		t.Fatalf("stdout = %q, want static target", stdout.String())
	}
}

func TestRunMenuRejectsForcedRichUIWithoutTerminal(t *testing.T) {
	t.Parallel()

	target := debianTarget()
	registry := provider.NewRegistry()
	if err := registry.Register(providerStub{targets: []provider.Target{target}}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	runner := app.New(app.Config{
		Registry: registry,
		Stdin:    strings.NewReader("1\n"),
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		Logger:   slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
		Mode:     app.ModeMenu,
		UIMode:   app.UIModeRich,
	})

	err := runner.Run(context.Background())
	if err == nil {
		t.Fatal("run app succeeded, want rich UI terminal error")
	}
	if !strings.Contains(err.Error(), "rich UI requires console") {
		t.Fatalf("run app error = %v, want rich UI console error", err)
	}
}

func TestRunRichMenuSelectsTargetThroughPTY(t *testing.T) {
	t.Parallel()

	targets := []provider.Target{debianTarget(), ubuntuTarget()}
	staged := provider.BootPlan{
		Target:  targets[1],
		Kernel:  provider.Artifact{Name: "linux", Path: "/tmp/bootup/linux"},
		Initrd:  provider.Artifact{Name: "initrd", Path: "/tmp/bootup/initrd"},
		Cmdline: "console=ttyS0",
	}

	registry := provider.NewRegistry()
	if err := registry.Register(providerStub{
		id:      "debian",
		targets: []provider.Target{targets[0]},
	}); err != nil {
		t.Fatalf("register Debian provider: %v", err)
	}
	if err := registry.Register(providerStub{
		id:      "ubuntu",
		targets: []provider.Target{targets[1]},
		plan:    provider.BootPlan{Target: targets[1]},
		staged:  staged,
	}); err != nil {
		t.Fatalf("register Ubuntu provider: %v", err)
	}

	console, err := expect.NewConsole(expect.WithDefaultTimeout(5 * time.Second))
	if err != nil {
		t.Fatalf("create console: %v", err)
	}
	if err := pty.Setsize(console.Tty(), &pty.Winsize{Rows: 25, Cols: 80}); err != nil {
		t.Fatalf("set console size: %v", err)
	}
	t.Cleanup(func() {
		if err := console.Close(); err != nil {
			t.Fatalf("close console: %v", err)
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	var executed provider.BootPlan
	errc := make(chan error, 1)
	go func() {
		var stderr bytes.Buffer
		runner := app.New(app.Config{
			Registry:   registry,
			Stdin:      console.Tty(),
			Stdout:     console.Tty(),
			Stderr:     &stderr,
			Logger:     slog.New(slog.NewTextHandler(&stderr, nil)),
			Mode:       app.ModeMenu,
			UIMode:     app.UIModeRich,
			StagingDir: t.TempDir(),
			Executor:   executorStub{executed: &executed},
		})
		errc <- runner.Run(ctx)
	}()

	if _, err := console.ExpectString("BOOTUP"); err != nil {
		t.Fatalf("expect rich menu banner: %v", err)
	}
	if _, err := console.ExpectString("Ubuntu 26.04 amd64 netboot"); err != nil {
		t.Fatalf("expect Ubuntu target: %v", err)
	}
	if _, err := console.Send("j\r"); err != nil {
		t.Fatalf("send rich menu selection: %v", err)
	}
	if _, err := console.ExpectString("PLANNING"); err != nil {
		t.Fatalf("expect rich planning status: %v", err)
	}

	select {
	case err := <-errc:
		if err != nil {
			t.Fatalf("run app: %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("run app timed out: %v", ctx.Err())
	}
	if executed.Target.ID != targets[1].ID {
		t.Fatalf("executed target = %q, want %q", executed.Target.ID, targets[1].ID)
	}
}

func TestRunBootRendersKexecFailure(t *testing.T) {
	t.Parallel()

	target := debianTarget()
	staged := provider.BootPlan{
		Target: target,
		Kernel: provider.Artifact{Name: "linux", Path: "/tmp/bootup/linux"},
		Initrd: provider.Artifact{Name: "initrd.gz", Path: "/tmp/bootup/initrd.gz"},
	}

	registry := provider.NewRegistry()
	if err := registry.Register(providerStub{
		targets: []provider.Target{target},
		plan:    provider.BootPlan{Target: target},
		staged:  staged,
	}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	var stdout bytes.Buffer
	runner := app.New(app.Config{
		Registry:   registry,
		Stdout:     &stdout,
		Stderr:     &bytes.Buffer{},
		Logger:     slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
		Mode:       app.ModeBootTarget,
		TargetID:   "debian-trixie-amd64-netboot",
		StagingDir: t.TempDir(),
		Executor:   executorStub{err: errors.New("operation not permitted")},
	})

	err := runner.Run(context.Background())
	if err == nil {
		t.Fatal("run app succeeded, want kexec error")
	}
	if !strings.Contains(stdout.String(), "bootup failure") {
		t.Fatalf("stdout = %q, want failure screen", stdout.String())
	}
	if !strings.Contains(stdout.String(), "operation not permitted") {
		t.Fatalf("stdout = %q, want executor error", stdout.String())
	}
}
