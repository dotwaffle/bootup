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
		ID:         "debian-trixie-amd64-netboot",
		ProviderID: "debian",
		Name:       "Debian trixie amd64 netboot installer with an intentionally long description",
		Catalog: provider.CatalogEntry{
			Architecture: "amd64",
			Distribution: "debian",
			Release:      "trixie",
			Kind:         "installer",
		},
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
	if !strings.Contains(out.String(), "debian/trixie/amd64/installer") {
		t.Fatalf("output = %q, want catalog label", out.String())
	}
}

func TestTextMenuRendersProgressAndFatalError(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	menu := ui.TextMenu{Width: 80}

	if err := menu.RenderStatus(&out, "verifying", "Debian metadata"); err != nil {
		t.Fatalf("render status: %v", err)
	}
	if err := menu.RenderFatal(&out, "signature validation failed"); err != nil {
		t.Fatalf("render fatal: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "[verifying] Debian metadata") {
		t.Fatalf("output = %q, want status", got)
	}
	if !strings.Contains(got, "signature validation failed") {
		t.Fatalf("output = %q, want fatal error", got)
	}
	if !strings.Contains(got, "bootup failure") {
		t.Fatalf("output = %q, want failure header", got)
	}
}

func TestTextMenuRendersDiscoveryFamilies(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	menu := ui.TextMenu{Width: 80}
	options := ui.BootOptions([]provider.Target{{
		ID:         "debian-trixie-amd64-netboot",
		ProviderID: "debian",
		Name:       "Debian trixie amd64 netboot",
		Catalog: provider.CatalogEntry{
			Distribution: "debian",
			Release:      "trixie",
			Architecture: "amd64",
			Kind:         "installer",
		},
	}}, []provider.DiscoveryFamily{{
		ID:         "debian",
		ProviderID: "debian",
		Name:       "Debian",
	}})

	if err := menu.RenderBootOptions(&out, options); err != nil {
		t.Fatalf("render options: %v", err)
	}

	got := out.String()
	for _, want := range []string{"debian-trixie-amd64-netboot", "discovery/debian", "Debian"} {
		if !strings.Contains(got, want) {
			t.Fatalf("output = %q, want %q", got, want)
		}
	}
}

func TestTextMenuRendersLifecycleDecoration(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	menu := ui.TextMenu{Width: 120}
	targets := []provider.Target{{
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
			Date:   "2028-06-01",
		},
	}}

	if err := menu.RenderTargets(&out, targets); err != nil {
		t.Fatalf("render targets: %v", err)
	}

	got := out.String()
	for _, want := range []string{"supported", "2028-06-01"} {
		if !strings.Contains(got, want) {
			t.Fatalf("output = %q, want lifecycle decoration %q", got, want)
		}
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

func TestSelectBootOptionByInputAcceptsFamilyID(t *testing.T) {
	t.Parallel()

	options := ui.BootOptions(nil, []provider.DiscoveryFamily{{
		ID:         "debian",
		ProviderID: "debian",
		Name:       "Debian",
	}})

	option, err := ui.SelectBootOptionByInput(options, "debian")
	if err != nil {
		t.Fatalf("select option: %v", err)
	}
	if option.Kind != ui.BootOptionDiscoveryFamily {
		t.Fatalf("option kind = %q, want discovery family", option.Kind)
	}
	if option.Family.ID != "debian" {
		t.Fatalf("family ID = %q, want debian", option.Family.ID)
	}
}

func TestSelectTargetByInputAcceptsIndexOrID(t *testing.T) {
	t.Parallel()

	targets := []provider.Target{
		{ID: "alpine", ProviderID: "alpine"},
		{ID: "debian-trixie-amd64-netboot", ProviderID: "debian"},
	}

	for _, input := range []string{"2", "debian-trixie-amd64-netboot"} {
		target, err := ui.SelectTargetByInput(targets, input)
		if err != nil {
			t.Fatalf("select target %q: %v", input, err)
		}
		if target.ID != "debian-trixie-amd64-netboot" {
			t.Fatalf("target ID = %q, want Debian", target.ID)
		}
	}
}
