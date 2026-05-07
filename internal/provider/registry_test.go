package provider_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/dotwaffle/bootup/internal/provider"
)

type testProvider struct {
	id      string
	targets []provider.Target
	plan    provider.BootPlan
	staged  provider.BootPlan
	planned *provider.PlanInput
}

func (p testProvider) ID() string {
	return p.id
}

func (p testProvider) Targets(context.Context) ([]provider.Target, error) {
	return p.targets, nil
}

func (p testProvider) Plan(_ context.Context, input provider.PlanInput) (provider.BootPlan, error) {
	if p.planned != nil {
		*p.planned = input
	}
	return p.plan, nil
}

func (p testProvider) Stage(context.Context, provider.StageConfig) (provider.BootPlan, error) {
	return p.staged, nil
}

type discoveryProviderStub struct {
	testProvider
	family        provider.DiscoveryFamily
	discovered    []provider.Target
	discoverErr   error
	discoverCalls *int
}

func (p discoveryProviderStub) DiscoveryFamily() provider.DiscoveryFamily {
	return p.family
}

func (p discoveryProviderStub) DiscoverTargets(context.Context) ([]provider.Target, error) {
	if p.discoverCalls != nil {
		*p.discoverCalls = *p.discoverCalls + 1
	}
	return p.discovered, p.discoverErr
}

func TestRegistryListsTargetsFromRegisteredProvider(t *testing.T) {
	t.Parallel()

	registry := provider.NewRegistry()
	target := provider.Target{
		ID:         "debian-trixie-amd64-netboot",
		ProviderID: "debian",
		Name:       "Debian trixie amd64 netboot",
		Catalog: provider.CatalogEntry{
			Architecture: "amd64",
			Distribution: "debian",
			Release:      "trixie",
			Kind:         "installer",
		},
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
	if !reflect.DeepEqual(targets[0], target) {
		t.Fatalf("target = %#v, want %#v", targets[0], target)
	}
}

func TestRegistryListsDiscoveryFamiliesWithoutDiscovery(t *testing.T) {
	t.Parallel()

	var discoverCalls int
	registry := provider.NewRegistry()
	if err := registry.Register(discoveryProviderStub{
		testProvider: testProvider{id: "debian"},
		family: provider.DiscoveryFamily{
			ID:         "debian",
			ProviderID: "debian",
			Name:       "Debian",
		},
		discoverCalls: &discoverCalls,
	}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	families, err := registry.DiscoveryFamilies()
	if err != nil {
		t.Fatalf("list discovery families: %v", err)
	}

	if discoverCalls != 0 {
		t.Fatalf("discovery calls = %d, want 0", discoverCalls)
	}
	if len(families) != 1 {
		t.Fatalf("families length = %d, want 1", len(families))
	}
	if families[0].ID != "debian" || families[0].ProviderID != "debian" {
		t.Fatalf("family = %#v, want Debian family", families[0])
	}
}

func TestRegistryDiscoversTargetsForSelectedFamily(t *testing.T) {
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
		Lifecycle: provider.LifecycleEntry{
			Status: provider.LifecycleSupported,
			Source: "debian",
		},
	}
	var discoverCalls int
	registry := provider.NewRegistry()
	if err := registry.Register(discoveryProviderStub{
		testProvider: testProvider{id: "debian"},
		family: provider.DiscoveryFamily{
			ID:         "debian",
			ProviderID: "debian",
			Name:       "Debian",
		},
		discovered:    []provider.Target{target},
		discoverCalls: &discoverCalls,
	}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	targets, err := registry.DiscoverTargets(context.Background(), "debian")
	if err != nil {
		t.Fatalf("discover targets: %v", err)
	}

	if discoverCalls != 1 {
		t.Fatalf("discovery calls = %d, want 1", discoverCalls)
	}
	if len(targets) != 1 || !reflect.DeepEqual(targets[0], target) {
		t.Fatalf("targets = %#v, want %#v", targets, []provider.Target{target})
	}
}

func TestRegistryRejectsUnknownDiscoveryFamily(t *testing.T) {
	t.Parallel()

	registry := provider.NewRegistry()

	_, err := registry.DiscoverTargets(context.Background(), "missing")
	if !errors.Is(err, provider.ErrDiscoveryFamilyNotFound) {
		t.Fatalf("discover targets error = %v, want %v", err, provider.ErrDiscoveryFamilyNotFound)
	}
}

func TestRegistryKeepsStaticTargetsWhenDiscoveryFails(t *testing.T) {
	t.Parallel()

	staticTarget := provider.Target{
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
	registry := provider.NewRegistry()
	if err := registry.Register(discoveryProviderStub{
		testProvider: testProvider{
			id:      "debian",
			targets: []provider.Target{staticTarget},
		},
		family: provider.DiscoveryFamily{
			ID:         "debian",
			ProviderID: "debian",
			Name:       "Debian",
		},
		discoverErr: errors.New("metadata unavailable"),
	}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	_, err := registry.DiscoverTargets(context.Background(), "debian")
	if err == nil {
		t.Fatal("discover targets succeeded, want error")
	}
	targets, err := registry.Targets(context.Background())
	if err != nil {
		t.Fatalf("list targets after discovery failure: %v", err)
	}
	if len(targets) != 1 || !reflect.DeepEqual(targets[0], staticTarget) {
		t.Fatalf("targets after discovery failure = %#v, want static target", targets)
	}
}

func TestRegistryRejectsInvalidProviderTargets(t *testing.T) {
	t.Parallel()

	validCatalog := provider.CatalogEntry{
		Distribution: "debian",
		Release:      "trixie",
		Architecture: "amd64",
		Kind:         "installer",
	}
	tests := []struct {
		name   string
		target provider.Target
	}{
		{
			name: "missing id",
			target: provider.Target{
				ProviderID: "debian",
				Name:       "Debian trixie amd64 netboot",
				Catalog:    validCatalog,
			},
		},
		{
			name: "mismatched provider id",
			target: provider.Target{
				ID:         "debian-trixie-amd64-netboot",
				ProviderID: "ubuntu",
				Name:       "Debian trixie amd64 netboot",
				Catalog:    validCatalog,
			},
		},
		{
			name: "missing display name",
			target: provider.Target{
				ID:         "debian-trixie-amd64-netboot",
				ProviderID: "debian",
				Catalog:    validCatalog,
			},
		},
		{
			name: "incomplete catalog",
			target: provider.Target{
				ID:         "debian-trixie-amd64-netboot",
				ProviderID: "debian",
				Name:       "Debian trixie amd64 netboot",
				Catalog: provider.CatalogEntry{
					Distribution: "debian",
					Release:      "trixie",
					Architecture: "amd64",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			registry := provider.NewRegistry()
			if err := registry.Register(testProvider{id: "debian", targets: []provider.Target{tt.target}}); err != nil {
				t.Fatalf("register provider: %v", err)
			}

			_, err := registry.Targets(context.Background())
			if !errors.Is(err, provider.ErrInvalidTarget) {
				t.Fatalf("targets error = %v, want %v", err, provider.ErrInvalidTarget)
			}
		})
	}
}

func TestRegistryAcceptsSupportedBootAction(t *testing.T) {
	t.Parallel()

	target := provider.Target{
		ID:         "local-disk-auto",
		ProviderID: "local",
		Name:       "Boot from local disk",
		Action:     provider.BootActionLocalBoot,
		Catalog: provider.CatalogEntry{
			Distribution: "local",
			Release:      "disk",
			Architecture: "amd64",
			Kind:         "localboot",
		},
	}

	registry := provider.NewRegistry()
	if err := registry.Register(testProvider{id: "local", targets: []provider.Target{target}}); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	targets, err := registry.Targets(context.Background())
	if err != nil {
		t.Fatalf("targets: %v", err)
	}
	if len(targets) != 1 || targets[0].Action != provider.BootActionLocalBoot {
		t.Fatalf("targets = %#v, want localboot action", targets)
	}
}

func TestRegistryRejectsInvalidBootAction(t *testing.T) {
	t.Parallel()

	target := provider.Target{
		ID:         "local-disk-auto",
		ProviderID: "local",
		Name:       "Boot from local disk",
		Action:     provider.BootAction("memdisk"),
		Catalog: provider.CatalogEntry{
			Distribution: "local",
			Release:      "disk",
			Architecture: "amd64",
			Kind:         "localboot",
		},
	}

	registry := provider.NewRegistry()
	if err := registry.Register(testProvider{id: "local", targets: []provider.Target{target}}); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	_, err := registry.Targets(context.Background())
	if !errors.Is(err, provider.ErrInvalidTarget) {
		t.Fatalf("targets error = %v, want %v", err, provider.ErrInvalidTarget)
	}
}

func TestRegistryValidatesStaticLinuxSource(t *testing.T) {
	t.Parallel()

	target := provider.Target{
		ID:         "opensuse-leap-160-amd64-netboot",
		ProviderID: "linux",
		Name:       "openSUSE Leap 16.0 amd64 installer",
		Catalog: provider.CatalogEntry{
			Distribution: "opensuse",
			Release:      "leap-16.0",
			Architecture: "amd64",
			Kind:         "installer",
		},
		Source: provider.SourceEntry{
			BaseURL:    "https://download.example/opensuse",
			KernelPath: "boot/x86_64/loader/linux",
			InitrdPath: "boot/x86_64/loader/initrd",
			Cmdline:    "install={base_url} console=ttyS0",
		},
	}

	registry := provider.NewRegistry()
	if err := registry.Register(testProvider{id: "linux", targets: []provider.Target{target}}); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	targets, err := registry.Targets(context.Background())
	if err != nil {
		t.Fatalf("targets: %v", err)
	}
	if targets[0].Source.KernelPath != "boot/x86_64/loader/linux" {
		t.Fatalf("kernel path = %q", targets[0].Source.KernelPath)
	}
}

func TestRegistryRejectsUnsafeStaticLinuxSourcePath(t *testing.T) {
	t.Parallel()

	target := provider.Target{
		ID:         "opensuse-leap-160-amd64-netboot",
		ProviderID: "linux",
		Name:       "openSUSE Leap 16.0 amd64 installer",
		Catalog: provider.CatalogEntry{
			Distribution: "opensuse",
			Release:      "leap-16.0",
			Architecture: "amd64",
			Kind:         "installer",
		},
		Source: provider.SourceEntry{
			BaseURL:    "https://download.example/opensuse",
			KernelPath: "../linux",
		},
	}

	registry := provider.NewRegistry()
	if err := registry.Register(testProvider{id: "linux", targets: []provider.Target{target}}); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	_, err := registry.Targets(context.Background())
	if !errors.Is(err, provider.ErrInvalidTarget) {
		t.Fatalf("targets error = %v, want %v", err, provider.ErrInvalidTarget)
	}
}

func TestTargetJSONOmitsZeroSource(t *testing.T) {
	t.Parallel()

	target := provider.Target{
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

	data, err := json.Marshal(target)
	if err != nil {
		t.Fatalf("marshal target: %v", err)
	}
	if string(data) != `{"id":"debian-trixie-amd64-netboot","provider_id":"debian","name":"Debian trixie amd64 netboot","catalog":{"distribution":"debian","release":"trixie","architecture":"amd64","kind":"installer"}}` {
		t.Fatalf("target JSON = %s", data)
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

	got, err := registry.Plan(context.Background(), provider.PlanInput{Target: target})
	if err != nil {
		t.Fatalf("plan target: %v", err)
	}
	if !reflect.DeepEqual(got, plan) {
		t.Fatalf("plan = %#v, want %#v", got, plan)
	}
}

func TestRegistryPassesSelectedOptionsToProviderPlan(t *testing.T) {
	t.Parallel()

	registry := provider.NewRegistry()
	target := targetWithOptions()
	var planned provider.PlanInput

	if err := registry.Register(testProvider{
		id:      "linux",
		targets: []provider.Target{target},
		plan:    provider.BootPlan{Target: target, Cmdline: "install=default"},
		planned: &planned,
	}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	options := []provider.SelectedOption{
		{ID: "text-install", Value: "true"},
		{ID: "mirror-url", Value: "https://mirror.example/opensuse"},
	}
	if _, err := registry.Plan(context.Background(), provider.PlanInput{Target: target, Options: options}); err != nil {
		t.Fatalf("plan target: %v", err)
	}
	if !reflect.DeepEqual(planned.Options, options) {
		t.Fatalf("planned options = %#v, want %#v", planned.Options, options)
	}
}

func TestRegistryRejectsInvalidSelectedOptionsBeforeProviderPlan(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		options []provider.SelectedOption
	}{
		{
			name:    "unknown option",
			options: []provider.SelectedOption{{ID: "missing", Value: "true"}},
		},
		{
			name:    "invalid bool value",
			options: []provider.SelectedOption{{ID: "text-install", Value: "sometimes"}},
		},
		{
			name:    "invalid enum value",
			options: []provider.SelectedOption{{ID: "serial-console", Value: "ttyS9"}},
		},
		{
			name:    "invalid string value",
			options: []provider.SelectedOption{{ID: "mirror-url", Value: " https://mirror.example/opensuse"}},
		},
		{
			name: "duplicate selection",
			options: []provider.SelectedOption{
				{ID: "text-install", Value: "true"},
				{ID: "text-install", Value: "false"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			registry := provider.NewRegistry()
			target := targetWithOptions()
			var planned provider.PlanInput
			if err := registry.Register(testProvider{
				id:      "linux",
				targets: []provider.Target{target},
				plan:    provider.BootPlan{Target: target},
				planned: &planned,
			}); err != nil {
				t.Fatalf("register provider: %v", err)
			}

			_, err := registry.Plan(context.Background(), provider.PlanInput{Target: target, Options: tt.options})
			if !errors.Is(err, provider.ErrInvalidTargetOption) {
				t.Fatalf("plan error = %v, want %v", err, provider.ErrInvalidTargetOption)
			}
			if planned.Target.ID != "" {
				t.Fatalf("provider planned target = %#v, want no provider call", planned.Target)
			}
		})
	}
}

func TestApplySelectedOptionsAppendsFragmentsInTargetOrder(t *testing.T) {
	t.Parallel()

	target := targetWithOptions()
	plan := provider.BootPlan{
		Target:  target,
		Cmdline: "install=default console=ttyS0",
	}
	selected := []provider.SelectedOption{
		{ID: "mirror-url", Value: "https://mirror.example/opensuse"},
		{ID: "serial-console", Value: "ttyS1"},
		{ID: "text-install", Value: "true"},
	}

	got, err := provider.ApplySelectedOptions(plan, selected)
	if err != nil {
		t.Fatalf("apply selected options: %v", err)
	}
	want := "install=default console=ttyS0 console=ttyS1 textmode=1 install=https://mirror.example/opensuse"
	if got.Cmdline != want {
		t.Fatalf("cmdline = %q, want %q", got.Cmdline, want)
	}
}

func TestApplySelectedOptionsAppendsFragmentsToFreeBSDKbootArgs(t *testing.T) {
	t.Parallel()

	target := provider.Target{
		ID:         "mfsbsd-142-amd64",
		ProviderID: "mfsbsd",
		Action:     provider.BootActionFreeBSDKboot,
		Options: []provider.TargetOption{
			{
				ID:       "hostname",
				Label:    "Hostname",
				Type:     provider.TargetOptionString,
				Template: "mfsbsd.hostname={value}",
			},
		},
	}
	plan := provider.BootPlan{
		Target: target,
		Action: provider.BootActionFreeBSDKboot,
		FreeBSDKboot: provider.FreeBSDKbootPlan{
			Args: []string{"hostfs_root=/tmp/mfsbsd-root"},
		},
	}

	got, err := provider.ApplySelectedOptions(plan, []provider.SelectedOption{
		{ID: "hostname", Value: "rescue-a"},
	})
	if err != nil {
		t.Fatalf("apply selected options: %v", err)
	}
	want := []string{"hostfs_root=/tmp/mfsbsd-root", "mfsbsd.hostname=rescue-a"}
	if !reflect.DeepEqual(got.FreeBSDKboot.Args, want) {
		t.Fatalf("freebsd kboot args = %#v, want %#v", got.FreeBSDKboot.Args, want)
	}
	if got.Cmdline != "" {
		t.Fatalf("cmdline = %q, want empty", got.Cmdline)
	}
}

func TestRegistryStagesThroughTargetProvider(t *testing.T) {
	t.Parallel()

	registry := provider.NewRegistry()
	target := provider.Target{ID: "debian-trixie-amd64-netboot", ProviderID: "debian"}
	staged := provider.BootPlan{
		Target: target,
		Kernel: provider.Artifact{Name: "linux", Path: "/tmp/linux"},
		Initrd: provider.Artifact{Name: "initrd.gz", Path: "/tmp/initrd.gz"},
	}

	if err := registry.Register(testProvider{id: "debian", staged: staged}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	got, err := registry.Stage(context.Background(), provider.StageConfig{
		Plan:       provider.BootPlan{Target: target},
		StagingDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("stage target: %v", err)
	}
	if !reflect.DeepEqual(got, staged) {
		t.Fatalf("staged plan = %#v, want %#v", got, staged)
	}
}

func targetWithOptions() provider.Target {
	return provider.Target{
		ID:         "opensuse-leap-160-amd64-netboot",
		ProviderID: "linux",
		Name:       "openSUSE Leap 16.0 amd64 installer",
		Catalog: provider.CatalogEntry{
			Distribution: "opensuse",
			Release:      "leap-16.0",
			Architecture: "amd64",
			Kind:         "installer",
		},
		Options: []provider.TargetOption{
			{
				ID:    "serial-console",
				Label: "Serial console",
				Type:  provider.TargetOptionEnum,
				Values: []provider.TargetOptionValue{
					{Value: "ttyS0", Fragment: "console=ttyS0"},
					{Value: "ttyS1", Fragment: "console=ttyS1"},
				},
			},
			{
				ID:       "text-install",
				Label:    "Text install",
				Type:     provider.TargetOptionBool,
				Fragment: "textmode=1",
			},
			{
				ID:       "mirror-url",
				Label:    "Installer mirror URL",
				Type:     provider.TargetOptionString,
				Template: "install={value}",
			},
		},
	}
}
