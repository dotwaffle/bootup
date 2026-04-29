package ui_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/dotwaffle/bootup/internal/provider"
	"github.com/dotwaffle/bootup/internal/ui"
)

func TestTextMenuRendersTargetsWithinSerialWidth(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	menu := ui.TextMenu{Width: 80}
	targets := []provider.Target{{
		ID:           "debian-trixie-amd64-netboot",
		ProviderID:   "debian",
		Name:         "Debian trixie amd64 netboot installer with an intentionally long description",
		Architecture: "amd64",
	}}

	if err := menu.RenderTargets(&out, targets); err != nil {
		t.Fatalf("render targets: %v", err)
	}

	for line := range strings.SplitSeq(strings.TrimRight(out.String(), "\n"), "\n") {
		if len(line) > 80 {
			t.Fatalf("line length = %d, want <= 80: %q", len(line), line)
		}
	}
	if !strings.Contains(out.String(), "debian-trixie-amd64-netboot") {
		t.Fatalf("output = %q, want target id", out.String())
	}
}

func TestTextMenuRendersProgressAndFatalError(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	menu := ui.TextMenu{Width: 80}

	if err := menu.RenderProgress(&out, "verifying Debian metadata"); err != nil {
		t.Fatalf("render progress: %v", err)
	}
	if err := menu.RenderFatal(&out, "signature validation failed"); err != nil {
		t.Fatalf("render fatal: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "verifying Debian metadata") {
		t.Fatalf("output = %q, want progress", got)
	}
	if !strings.Contains(got, "signature validation failed") {
		t.Fatalf("output = %q, want fatal error", got)
	}
}

func TestSelectTargetByID(t *testing.T) {
	t.Parallel()

	targets := []provider.Target{
		{ID: "alpine", ProviderID: "alpine"},
		{ID: "debian-trixie-amd64-netboot", ProviderID: "debian"},
	}

	target, err := ui.SelectTargetByID(targets, "debian-trixie-amd64-netboot")
	if err != nil {
		t.Fatalf("select target: %v", err)
	}
	if target.ID != "debian-trixie-amd64-netboot" {
		t.Fatalf("target ID = %q, want Debian", target.ID)
	}
}
