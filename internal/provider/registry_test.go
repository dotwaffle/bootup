package provider_test

import (
	"context"
	"errors"
	"testing"

	"github.com/dotwaffle/bootup/internal/provider"
)

type testProvider struct {
	id      string
	targets []provider.Target
	plan    provider.BootPlan
}

func (p testProvider) ID() string {
	return p.id
}

func (p testProvider) Targets(context.Context) ([]provider.Target, error) {
	return p.targets, nil
}

func (p testProvider) Plan(context.Context, provider.Target) (provider.BootPlan, error) {
	return p.plan, nil
}

func TestRegistryListsTargetsFromRegisteredProvider(t *testing.T) {
	t.Parallel()

	registry := provider.NewRegistry()
	target := provider.Target{
		ID:           "debian-trixie-amd64-netboot",
		ProviderID:   "debian",
		Name:         "Debian trixie amd64 netboot",
		Architecture: "amd64",
	}

	if err := registry.Register(testProvider{id: "debian", targets: []provider.Target{target}}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	targets, err := registry.Targets(context.Background())
	if err != nil {
		t.Fatalf("list targets: %v", err)
	}

	if len(targets) != 1 {
		t.Fatalf("targets length = %d, want 1", len(targets))
	}
	if targets[0] != target {
		t.Fatalf("target = %#v, want %#v", targets[0], target)
	}
}

func TestRegistryRejectsDuplicateProviderID(t *testing.T) {
	t.Parallel()

	registry := provider.NewRegistry()
	if err := registry.Register(testProvider{id: "debian"}); err != nil {
		t.Fatalf("register first provider: %v", err)
	}

	err := registry.Register(testProvider{id: "debian"})
	if !errors.Is(err, provider.ErrDuplicateProvider) {
		t.Fatalf("duplicate provider error = %v, want %v", err, provider.ErrDuplicateProvider)
	}
}

func TestRegistryPlansThroughTargetProvider(t *testing.T) {
	t.Parallel()

	registry := provider.NewRegistry()
	target := provider.Target{ID: "debian-trixie-amd64-netboot", ProviderID: "debian"}
	plan := provider.BootPlan{
		Target:  target,
		Kernel:  provider.Artifact{Name: "linux", URL: "https://example.test/linux"},
		Initrd:  provider.Artifact{Name: "initrd.gz", URL: "https://example.test/initrd.gz"},
		Cmdline: "priority=low",
	}

	if err := registry.Register(testProvider{id: "debian", targets: []provider.Target{target}, plan: plan}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	got, err := registry.Plan(context.Background(), target)
	if err != nil {
		t.Fatalf("plan target: %v", err)
	}
	if got != plan {
		t.Fatalf("plan = %#v, want %#v", got, plan)
	}
}
