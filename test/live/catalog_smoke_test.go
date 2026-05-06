package live_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dotwaffle/bootup/internal/catalog"
	"github.com/dotwaffle/bootup/internal/provider"
	"github.com/dotwaffle/bootup/internal/providers/linux"
	"github.com/dotwaffle/bootup/internal/ui"
)

const liveCatalogSmokeEnv = "BOOTUP_LIVE_CATALOG_SMOKE"

var errUnsupportedCatalogSmokeTarget = errors.New("unsupported live smoke target")

func TestCatalogLiveSmokeSelectsTargetsByID(t *testing.T) {
	t.Parallel()

	memtest := catalogSmokeTarget(t, "memtest86plus-800-amd64")
	if err := requireCatalogSmokeSupported(memtest); err != nil {
		t.Fatalf("MemTest86+ support: %v", err)
	}
	if memtest.Source.InitrdPath != "" {
		t.Fatalf("MemTest86+ initrd path = %q, want kernel-only target", memtest.Source.InitrdPath)
	}

	opensuse := catalogSmokeTarget(t, "opensuse-leap-160-amd64-netboot")
	if err := requireCatalogSmokeSupported(opensuse); err != nil {
		t.Fatalf("openSUSE support: %v", err)
	}
	if opensuse.Source.KernelPath == "" || opensuse.Source.InitrdPath == "" {
		t.Fatalf("openSUSE source = %#v, want kernel and initrd paths", opensuse.Source)
	}
}

func TestCatalogLiveSmokeReportsUnsupportedTargets(t *testing.T) {
	t.Parallel()

	target := catalogSmokeTarget(t, "local-disk-auto")
	err := requireCatalogSmokeSupported(target)
	if !errors.Is(err, errUnsupportedCatalogSmokeTarget) {
		t.Fatalf("support error = %v, want %v", err, errUnsupportedCatalogSmokeTarget)
	}
}

func TestLiveCatalogStagesMemTest86PlusKernelOnly(t *testing.T) {
	if os.Getenv(liveCatalogSmokeEnv) != "1" {
		t.Skip(liveCatalogSmokeEnv + "=1 is required")
	}

	staged := stageLiveCatalogTarget(t, "memtest86plus-800-amd64")
	if staged.Initrd != (provider.Artifact{}) {
		t.Fatalf("initrd = %#v, want none", staged.Initrd)
	}
	assertNonEmptyArtifact(t, staged.Kernel.Path)
	if filepath.Base(staged.Kernel.Path) != "mt86p_800_x86_64" {
		t.Fatalf("kernel path = %q, want mt86p_800_x86_64", staged.Kernel.Path)
	}
}

func TestLiveCatalogStagesGenericLinuxKernelAndInitrd(t *testing.T) {
	if os.Getenv(liveCatalogSmokeEnv) != "1" {
		t.Skip(liveCatalogSmokeEnv + "=1 is required")
	}

	staged := stageLiveCatalogTarget(t, "opensuse-leap-160-amd64-netboot")
	assertNonEmptyArtifact(t, staged.Kernel.Path)
	assertNonEmptyArtifact(t, staged.Initrd.Path)
	if filepath.Base(staged.Kernel.Path) != "linux" {
		t.Fatalf("kernel path = %q, want linux", staged.Kernel.Path)
	}
	if filepath.Base(staged.Initrd.Path) != "initrd" {
		t.Fatalf("initrd path = %q, want initrd", staged.Initrd.Path)
	}
}

func stageLiveCatalogTarget(t *testing.T, targetID string) provider.BootPlan {
	t.Helper()

	target := catalogSmokeTarget(t, targetID)
	if err := requireCatalogSmokeSupported(target); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)

	p := linux.NewProvider(linux.Config{Targets: catalogSmokeTargets(t, "linux")})
	plan, err := p.Plan(ctx, provider.PlanInput{Target: target})
	if err != nil {
		t.Fatalf("plan catalog target %s: %v", targetID, err)
	}
	staged, err := p.Stage(ctx, provider.StageConfig{
		Plan:       plan,
		StagingDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("stage live catalog target %s: %v", targetID, err)
	}
	return staged
}

func catalogSmokeTarget(t *testing.T, targetID string) provider.Target {
	t.Helper()

	targets := catalogSmokeTargets(t, "debian", "ubuntu", "fedora", "linux", "local")
	target, err := ui.SelectTargetByID(targets, targetID)
	if err != nil {
		t.Fatalf("select catalog target %s: %v", targetID, err)
	}
	return target
}

func catalogSmokeTargets(t *testing.T, providerIDs ...string) []provider.Target {
	t.Helper()

	doc, err := catalog.LoadDefault([]string{"debian", "ubuntu", "fedora", "linux", "local"})
	if err != nil {
		t.Fatalf("load default catalog: %v", err)
	}
	var targets []provider.Target
	for _, providerID := range providerIDs {
		targets = append(targets, doc.Targets(providerID)...)
	}
	return targets
}

func requireCatalogSmokeSupported(target provider.Target) error {
	if provider.ResolveBootAction(target.Action) != provider.BootActionLinuxKexec {
		return errUnsupportedCatalogSmokeTarget
	}
	if target.ProviderID != "linux" {
		return errUnsupportedCatalogSmokeTarget
	}
	if target.Source.BaseURL == "" || target.Source.KernelPath == "" {
		return errUnsupportedCatalogSmokeTarget
	}
	return nil
}

func assertNonEmptyArtifact(t *testing.T, path string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read staged artifact %s: %v", path, err)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		t.Fatalf("staged artifact %s is empty", path)
	}
}
