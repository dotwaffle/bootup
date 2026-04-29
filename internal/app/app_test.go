package app_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/dotwaffle/bootup/internal/app"
	"github.com/dotwaffle/bootup/internal/provider"
)

type providerStub struct {
	targets []provider.Target
	plan    provider.BootPlan
	planned *provider.Target
}

func (p providerStub) ID() string {
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

func TestRunListsTargetsInNonInteractiveMode(t *testing.T) {
	t.Parallel()

	registry := provider.NewRegistry()
	if err := registry.Register(providerStub{targets: []provider.Target{{
		ID:           "debian-trixie-amd64-netboot",
		ProviderID:   "debian",
		Name:         "Debian trixie amd64 netboot",
		Architecture: "amd64",
	}}}); err != nil {
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

	if !strings.Contains(stdout.String(), "Debian trixie amd64 netboot") {
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
	if err := registry.Register(providerStub{targets: []provider.Target{{
		ID:         "debian-trixie-amd64-netboot",
		ProviderID: "debian",
		Name:       "Debian trixie amd64 netboot",
	}}}); err != nil {
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

func TestRunPlansSelectedTargetInNonInteractiveMode(t *testing.T) {
	t.Parallel()

	target := provider.Target{
		ID:         "debian-trixie-amd64-netboot",
		ProviderID: "debian",
		Name:       "Debian trixie amd64 netboot",
	}
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
