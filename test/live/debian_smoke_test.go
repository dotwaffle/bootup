package live_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dotwaffle/bootup/internal/provider"
	"github.com/dotwaffle/bootup/internal/providers/debian"
)

func TestLiveDebianStagesInstallerArtifacts(t *testing.T) {
	if os.Getenv("BOOTUP_LIVE_DEBIAN_SMOKE") != "1" {
		t.Skip("BOOTUP_LIVE_DEBIAN_SMOKE=1 is required")
	}

	const keyringPath = "/usr/share/keyrings/debian-archive-keyring.gpg"
	keyring, err := os.ReadFile(keyringPath)
	if err != nil {
		t.Skipf("read Debian archive keyring %s: %v", keyringPath, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)

	p := debian.NewProvider(debian.Config{Keyring: keyring})
	targets, err := p.Targets(ctx)
	if err != nil {
		t.Fatalf("targets: %v", err)
	}
	plan, err := p.Plan(ctx, provider.PlanInput{Target: targets[0]})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	staged, err := p.Stage(ctx, provider.StageConfig{
		Plan:       plan,
		StagingDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("stage live Debian artifacts: %v", err)
	}

	for _, path := range []string{staged.Kernel.Path, staged.Initrd.Path} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read staged artifact %s: %v", path, err)
		}
		if len(bytes.TrimSpace(data)) == 0 {
			t.Fatalf("staged artifact %s is empty", path)
		}
	}
	if filepath.Base(staged.Kernel.Path) != "linux" {
		t.Fatalf("kernel path = %q, want linux", staged.Kernel.Path)
	}
	if filepath.Base(staged.Initrd.Path) != "initrd.gz" {
		t.Fatalf("initrd path = %q, want initrd.gz", staged.Initrd.Path)
	}
}
