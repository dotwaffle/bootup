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

func TestTextMenuRendersCatalogListMetadata(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	menu := ui.TextMenu{Width: 120}
	targets := []provider.Target{{
		ID:         "opensuse-leap-160-amd64-netboot",
		ProviderID: "linux",
		Name:       "openSUSE Leap 16.0 amd64 installer",
		Action:     provider.BootActionLinuxKexec,
		Catalog: provider.CatalogEntry{
			Distribution: "opensuse",
			Release:      "leap-16.0",
			Architecture: "amd64",
			Kind:         "installer",
		},
	}}

	if err := menu.RenderTargets(&out, targets); err != nil {
		t.Fatalf("render targets: %v", err)
	}

	got := out.String()
	for _, want := range []string{
		"opensuse-leap-160-amd64-netboot",
		"openSUSE Leap 16.0 amd64 installer",
		"distribution=opensuse",
		"release=leap-16.0",
		"architecture=amd64",
		"provider=linux",
		"action=linux-kexec",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("output = %q, want %q", got, want)
		}
	}
}

func TestTextMenuRendersTargetDetails(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	menu := ui.TextMenu{Width: 120}
	target := provider.Target{
		ID:         "opensuse-leap-160-amd64-netboot",
		ProviderID: "linux",
		Name:       "openSUSE Leap 16.0 amd64 installer",
		Action:     provider.BootActionLinuxKexec,
		Catalog: provider.CatalogEntry{
			Distribution: "opensuse",
			Release:      "leap-16.0",
			Architecture: "amd64",
			Kind:         "installer",
		},
		Source: provider.SourceEntry{
			BaseURL:    "https://download.example/opensuse",
			ISOName:    "opensuse.iso",
			ISOSHA256:  strings.Repeat("a", 64),
			KernelPath: "boot/x86_64/loader/linux",
			InitrdPath: "boot/x86_64/loader/initrd",
			Cmdline:    "install={base_url}",
		},
		Lifecycle: provider.LifecycleEntry{
			Status: provider.LifecycleSupported,
			Source: "catalog",
		},
		Options: []provider.TargetOption{
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
		Secrets: []provider.SecretInput{{
			ID:       "installer-password",
			Label:    "Installer password",
			Purpose:  "Used by the installer automation profile.",
			Required: true,
			Delivery: provider.SecretDeliveryStagedFile,
		}},
	}

	if err := menu.RenderTargetDetails(&out, target); err != nil {
		t.Fatalf("render target details: %v", err)
	}

	got := out.String()
	for _, want := range []string{
		"id: opensuse-leap-160-amd64-netboot",
		"name: openSUSE Leap 16.0 amd64 installer",
		"provider: linux",
		"action: linux-kexec",
		"distribution: opensuse",
		"base_url: https://download.example/opensuse",
		"iso_name: opensuse.iso",
		"iso_sha256: " + strings.Repeat("a", 64),
		"kernel_path: boot/x86_64/loader/linux",
		"lifecycle: supported catalog",
		"options:",
		"text-install bool Text install fragment=textmode=1",
		"mirror-url string Installer mirror URL template=install={value}",
		"secrets:",
		"installer-password required staged-file Installer password purpose=Used by the installer automation profile.",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("output = %q, want %q", got, want)
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
