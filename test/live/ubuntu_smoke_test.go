package live_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dotwaffle/bootup/internal/provider"
	"github.com/dotwaffle/bootup/internal/providers/ubuntu"
)

func TestLiveUbuntuStagesInstallerArtifacts(t *testing.T) {
	if os.Getenv("BOOTUP_LIVE_UBUNTU_SMOKE") != "1" {
		t.Skip("BOOTUP_LIVE_UBUNTU_SMOKE=1 is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)

	p := ubuntu.NewProvider(ubuntu.Config{})
	targets, err := p.Targets(ctx)
	if err != nil {
		t.Fatalf("targets: %v", err)
	}
	plan, err := p.Plan(ctx, targets[0])
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	staged, err := p.Stage(ctx, provider.StageConfig{
		Plan:       plan,
		StagingDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("stage live Ubuntu artifacts: %v", err)
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
	if filepath.Base(staged.Initrd.Path) != "initrd" {
		t.Fatalf("initrd path = %q, want initrd", staged.Initrd.Path)
	}
}
