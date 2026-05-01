package localdisk_test

import (
	"context"
	"testing"

	"github.com/dotwaffle/bootup/internal/provider"
	"github.com/dotwaffle/bootup/internal/providers/localdisk"
)

func TestProviderPlansLocalBootAction(t *testing.T) {
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
	p := localdisk.NewProvider(localdisk.Config{Targets: []provider.Target{target}})

	plan, err := p.Plan(context.Background(), target)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if plan.Action != provider.BootActionLocalBoot {
		t.Fatalf("plan action = %q, want localboot", plan.Action)
	}
	if plan.Kernel != (provider.Artifact{}) || plan.Initrd != (provider.Artifact{}) {
		t.Fatalf("plan artifacts = %#v/%#v, want none", plan.Kernel, plan.Initrd)
	}

	staged, err := p.Stage(context.Background(), provider.StageConfig{Plan: plan, StagingDir: t.TempDir()})
	if err != nil {
		t.Fatalf("stage: %v", err)
	}
	if staged != plan {
		t.Fatalf("staged plan = %#v, want original plan", staged)
	}
}
