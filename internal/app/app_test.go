package app_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"reflect"
	"strings"
	"testing"
	"time"

	expect "github.com/Netflix/go-expect"
	"github.com/creack/pty"
	"github.com/dotwaffle/bootup/internal/app"
	"github.com/dotwaffle/bootup/internal/provider"
)

type providerStub struct {
	id             string
	targets        []provider.Target
	plan           provider.BootPlan
	staged         provider.BootPlan
	planned        *provider.PlanInput
	stageConfig    *provider.StageConfig
	stageInputPlan bool
	applyOptions   bool
	planErr        error
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

func (p providerStub) Plan(_ context.Context, input provider.PlanInput) (provider.BootPlan, error) {
	if p.planned != nil {
		*p.planned = input
	}
	if p.planErr != nil {
		return provider.BootPlan{}, p.planErr
	}
	if p.applyOptions {
		return provider.ApplySelectedOptions(p.plan, input.Options)
	}
	return p.plan, nil
}

func (p providerStub) Stage(_ context.Context, config provider.StageConfig) (provider.BootPlan, error) {
	if p.stageConfig != nil {
		*p.stageConfig = config
	}
	if p.stageInputPlan {
		return config.Plan, nil
	}
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

func mfsBSDTarget() provider.Target {
	return provider.Target{
		ID:         "mfsbsd-142-amd64",
		ProviderID: "mfsbsd",
		Name:       "mfsBSD 14.2 amd64",
		Action:     provider.BootActionFreeBSDKboot,
		Catalog: provider.CatalogEntry{
			Distribution: "mfsbsd",
			Release:      "14.2",
			Architecture: "amd64",
			Kind:         "rescue",
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

func TestRunShowsSelectedTargetDetails(t *testing.T) {
	t.Parallel()

	target := debianTarget()
	target.Source = provider.SourceEntry{
		BaseURL:    "https://mirror.example/debian",
		KernelPath: "netboot/linux",
		InitrdPath: "netboot/initrd.gz",
	}
	target.Lifecycle = provider.LifecycleEntry{
		Status: provider.LifecycleSupported,
		Source: "catalog",
	}
	target.Options = []provider.TargetOption{{
		ID:       "text-install",
		Label:    "Text install",
		Type:     provider.TargetOptionBool,
		Fragment: "textmode=1",
	}}

	registry := provider.NewRegistry()
	if err := registry.Register(providerStub{targets: []provider.Target{target}}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	var stdout bytes.Buffer
	runner := app.New(app.Config{
		Registry: registry,
		Stdout:   &stdout,
		Stderr:   &bytes.Buffer{},
		Logger:   slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
		Mode:     app.ModeShowTarget,
		TargetID: "debian-trixie-amd64-netboot",
	})

	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("run app: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{
		"id: debian-trixie-amd64-netboot",
		"provider: debian",
		"base_url: https://mirror.example/debian",
		"lifecycle: supported catalog",
		"text-install bool Text install fragment=textmode=1",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("stdout = %q, want %q", got, want)
		}
	}
}

func TestRunRendersCatalogMatrix(t *testing.T) {
	t.Parallel()

	target := debianTarget()
	registry := provider.NewRegistry()
	if err := registry.Register(providerStub{
		targets: []provider.Target{target},
		plan: provider.BootPlan{
			Target: target,
			Kernel: provider.Artifact{URL: "https://mirror.example/linux"},
			Initrd: provider.Artifact{URL: "https://mirror.example/initrd.gz"},
		},
	}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	var stdout bytes.Buffer
	runner := app.New(app.Config{
		Registry: registry,
		Stdout:   &stdout,
		Stderr:   &bytes.Buffer{},
		Logger:   slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
		Mode:     app.ModeCatalogMatrix,
	})

	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("run app: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{
		"bootup catalog matrix",
		"target\tprovider\tdistribution\trelease\tarchitecture\tkind\tlifecycle\taction\tplan\ttrust\tsmoke\terror",
		catalogMatrixTestRow("debian-trixie-amd64-netboot", "debian", "debian", "trixie", "amd64", "installer", "", "linux-kexec", "ok", "https-only", "debian-qemu", ""),
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("stdout = %q, want %q", got, want)
		}
	}
}

func TestRunCatalogMatrixReportsPlanErrors(t *testing.T) {
	t.Parallel()

	target := debianTarget()
	registry := provider.NewRegistry()
	if err := registry.Register(providerStub{
		targets: []provider.Target{target},
		planErr: errors.New("provider cannot plan target"),
	}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	var stdout bytes.Buffer
	runner := app.New(app.Config{
		Registry: registry,
		Stdout:   &stdout,
		Stderr:   &bytes.Buffer{},
		Logger:   slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
		Mode:     app.ModeCatalogMatrix,
	})

	err := runner.Run(context.Background())
	if err == nil {
		t.Fatal("run app succeeded, want catalog matrix error")
	}
	if !strings.Contains(err.Error(), "catalog matrix has 1 planning error") {
		t.Fatalf("run error = %v, want planning error count", err)
	}
	got := stdout.String()
	for _, want := range []string{
		catalogMatrixTestRow("debian-trixie-amd64-netboot", "debian", "debian", "trixie", "amd64", "installer", "", "linux-kexec", "error", "unknown", "debian-qemu", ""),
		"provider cannot plan target",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("stdout = %q, want %q", got, want)
		}
	}
}

func catalogMatrixTestRow(fields ...string) string {
	return strings.Join(fields, "\t")
}

func TestRunShowTargetRejectsUnknownWithoutPlanning(t *testing.T) {
	t.Parallel()

	var planned provider.PlanInput
	var stageConfig provider.StageConfig
	registry := provider.NewRegistry()
	if err := registry.Register(providerStub{
		targets:     []provider.Target{debianTarget()},
		planned:     &planned,
		stageConfig: &stageConfig,
	}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	runner := app.New(app.Config{
		Registry: registry,
		Stdout:   &bytes.Buffer{},
		Stderr:   &bytes.Buffer{},
		Logger:   slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
		Mode:     app.ModeShowTarget,
		TargetID: "missing-target",
	})

	err := runner.Run(context.Background())
	if err == nil {
		t.Fatal("run app succeeded, want unknown target error")
	}
	if !strings.Contains(err.Error(), `target "missing-target" not found`) {
		t.Fatalf("run error = %v, want unknown target", err)
	}
	if planned.Target.ID != "" {
		t.Fatalf("planned target = %#v, want no planning", planned.Target)
	}
	if stageConfig.StagingDir != "" {
		t.Fatalf("stage config = %#v, want no staging", stageConfig)
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
	var planned provider.PlanInput

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
	if planned.Target.ID != target.ID {
		t.Fatalf("planned target = %#v, want %#v", planned, target)
	}
	if !strings.Contains(stdout.String(), "https://example.test/linux") {
		t.Fatalf("stdout = %q, want kernel URL", stdout.String())
	}
}

func TestRunPlansFreeBSDKbootTargetInNonInteractiveMode(t *testing.T) {
	t.Parallel()

	target := mfsBSDTarget()
	plan := provider.BootPlan{
		Target: target,
		Action: provider.BootActionFreeBSDKboot,
		FreeBSDKboot: provider.FreeBSDKbootPlan{
			Payload: provider.Artifact{
				Name: "mfsbsd.iso",
				URL:  "https://mfsbsd.example/mfsbsd.iso",
			},
			LoaderArchive: provider.Artifact{
				Name: "base.txz",
				URL:  "https://download.example/base.txz",
			},
		},
	}

	registry := provider.NewRegistry()
	if err := registry.Register(providerStub{
		id:      "mfsbsd",
		targets: []provider.Target{target},
		plan:    plan,
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
		TargetID: "mfsbsd-142-amd64",
	})

	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("run app: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{
		"loader_archive\thttps://download.example/base.txz",
		"payload\thttps://mfsbsd.example/mfsbsd.iso",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("stdout = %q, want %q", got, want)
		}
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

func TestRunStagesFreeBSDKbootTargetInNonInteractiveMode(t *testing.T) {
	t.Parallel()

	target := mfsBSDTarget()
	plan := provider.BootPlan{Target: target, Action: provider.BootActionFreeBSDKboot}
	staged := provider.BootPlan{
		Target: target,
		Action: provider.BootActionFreeBSDKboot,
		FreeBSDKboot: provider.FreeBSDKbootPlan{
			Loader:      provider.Artifact{Name: "loader.kboot", Path: "/tmp/bootup/freebsd-kboot/loader.kboot"},
			LoaderHelp:  provider.Artifact{Name: "loader.help.kboot", Path: "/tmp/bootup/freebsd-kboot/loader.help.kboot"},
			Payload:     provider.Artifact{Name: "mfsbsd.iso", Path: "/tmp/bootup/mfsbsd.iso"},
			PayloadRoot: "/tmp/bootup/mfsbsd-root",
			Args:        []string{"hostfs_root=/tmp/bootup/mfsbsd-root", "bootdev=host:/"},
		},
	}

	registry := provider.NewRegistry()
	if err := registry.Register(providerStub{
		id:      "mfsbsd",
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
		TargetID:   "mfsbsd-142-amd64",
		StagingDir: t.TempDir(),
	})

	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("run app: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{
		"loader\t/tmp/bootup/freebsd-kboot/loader.kboot",
		"payload_root\t/tmp/bootup/mfsbsd-root",
		"args\thostfs_root=/tmp/bootup/mfsbsd-root bootdev=host:/",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("stdout = %q, want %q", got, want)
		}
	}
}

func TestRunAppendsCmdlineBeforeStaging(t *testing.T) {
	t.Parallel()

	target := debianTarget()
	plan := provider.BootPlan{
		Target:  target,
		Kernel:  provider.Artifact{Name: "linux", URL: "https://example.test/linux"},
		Initrd:  provider.Artifact{Name: "initrd.gz", URL: "https://example.test/initrd.gz"},
		Cmdline: "priority=low",
	}
	var stageConfig provider.StageConfig

	registry := provider.NewRegistry()
	if err := registry.Register(providerStub{
		targets:        []provider.Target{target},
		plan:           plan,
		stageConfig:    &stageConfig,
		stageInputPlan: true,
	}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	runner := app.New(app.Config{
		Registry:      registry,
		Stdout:        &bytes.Buffer{},
		Stderr:        &bytes.Buffer{},
		Logger:        slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
		Mode:          app.ModeStageTarget,
		TargetID:      "debian-trixie-amd64-netboot",
		StagingDir:    t.TempDir(),
		CmdlineAppend: "inst.vnc console=ttyS1",
	})

	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("run app: %v", err)
	}
	if stageConfig.Plan.Cmdline != "priority=low inst.vnc console=ttyS1" {
		t.Fatalf("staged cmdline = %q", stageConfig.Plan.Cmdline)
	}
}

func TestRunPassesTargetOptionsAndAppendsCmdlineLast(t *testing.T) {
	t.Parallel()

	target := debianTarget()
	target.Options = []provider.TargetOption{
		{
			ID:       "text-install",
			Label:    "Text install",
			Type:     provider.TargetOptionBool,
			Fragment: "textmode=1",
		},
		{
			ID:       "mirror-url",
			Label:    "Mirror URL",
			Type:     provider.TargetOptionString,
			Template: "install={value}",
		},
	}
	plan := provider.BootPlan{
		Target:  target,
		Cmdline: "priority=low",
	}
	selectedOptions := []provider.SelectedOption{
		{ID: "mirror-url", Value: "https://mirror.example/debian"},
		{ID: "text-install", Value: "true"},
	}
	var planned provider.PlanInput
	var stageConfig provider.StageConfig

	registry := provider.NewRegistry()
	if err := registry.Register(providerStub{
		targets:        []provider.Target{target},
		plan:           plan,
		planned:        &planned,
		stageConfig:    &stageConfig,
		stageInputPlan: true,
		applyOptions:   true,
	}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	runner := app.New(app.Config{
		Registry:      registry,
		Stdout:        &bytes.Buffer{},
		Stderr:        &bytes.Buffer{},
		Logger:        slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
		Mode:          app.ModeStageTarget,
		TargetID:      "debian-trixie-amd64-netboot",
		TargetOptions: selectedOptions,
		StagingDir:    t.TempDir(),
		CmdlineAppend: "console=ttyS1",
	})

	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("run app: %v", err)
	}
	if !reflect.DeepEqual(planned.Options, selectedOptions) {
		t.Fatalf("planned options = %#v, want %#v", planned.Options, selectedOptions)
	}
	wantCmdline := "priority=low textmode=1 install=https://mirror.example/debian console=ttyS1"
	if stageConfig.Plan.Cmdline != wantCmdline {
		t.Fatalf("staged cmdline = %q, want %q", stageConfig.Plan.Cmdline, wantCmdline)
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
	if raceDetectorEnabled {
		t.Skip("Bubble Tea's Linux cancelreader races during PTY shutdown under -race")
	}

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
