package catalog_test

import (
	"bytes"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/dotwaffle/bootup/internal/catalog"
)

func TestParseValidatesAndFiltersStaticCatalog(t *testing.T) {
	t.Parallel()

	doc, err := catalog.Parse([]byte(`{
		"schema_version": 1,
		"targets": [
			{
				"id": "debian-bookworm-amd64-netboot",
				"provider_id": "debian",
				"name": "Debian bookworm amd64 netboot",
				"catalog": {
					"distribution": "debian",
					"release": "bookworm",
					"architecture": "amd64",
					"kind": "installer"
				}
			},
			{
				"id": "debian-trixie-amd64-netboot",
				"provider_id": "debian",
				"name": "Debian trixie amd64 netboot",
				"catalog": {
					"distribution": "debian",
					"release": "trixie",
					"architecture": "amd64",
					"kind": "installer"
				}
			},
			{
				"id": "ubuntu-2604-amd64-netboot",
				"provider_id": "ubuntu",
				"name": "Ubuntu 26.04 amd64 netboot",
				"catalog": {
					"distribution": "ubuntu",
					"release": "26.04",
					"architecture": "amd64",
					"kind": "installer"
				},
				"source": {
					"base_url": "https://releases.example/26.04",
					"iso_name": "ubuntu-26.04-live-server-amd64.iso"
				}
			}
		]
	}`), []string{"debian", "ubuntu"})
	if err != nil {
		t.Fatalf("parse catalog: %v", err)
	}

	debianTargets := doc.Targets("debian")
	if len(debianTargets) != 2 {
		t.Fatalf("Debian targets length = %d, want 2", len(debianTargets))
	}
	if debianTargets[0].ID != "debian-bookworm-amd64-netboot" {
		t.Fatalf("first Debian target = %q, want bookworm", debianTargets[0].ID)
	}
	debianTargets[0].ID = "mutated"
	if got := doc.Targets("debian")[0].ID; got != "debian-bookworm-amd64-netboot" {
		t.Fatalf("document targets were mutated to %q", got)
	}

	ubuntuTargets := doc.Targets("ubuntu")
	if len(ubuntuTargets) != 1 || ubuntuTargets[0].ID != "ubuntu-2604-amd64-netboot" {
		t.Fatalf("Ubuntu targets = %#v, want 26.04 target", ubuntuTargets)
	}
	if ubuntuTargets[0].Source.BaseURL != "https://releases.example/26.04" {
		t.Fatalf("Ubuntu source base URL = %q", ubuntuTargets[0].Source.BaseURL)
	}
	if ubuntuTargets[0].Source.ISOName != "ubuntu-26.04-live-server-amd64.iso" {
		t.Fatalf("Ubuntu source ISO name = %q", ubuntuTargets[0].Source.ISOName)
	}
}

func TestLoadDefaultIncludesInitialStaticTargets(t *testing.T) {
	t.Parallel()

	doc, err := catalog.LoadDefault([]string{"debian", "ubuntu", "fedora"})
	if err != nil {
		t.Fatalf("load default catalog: %v", err)
	}

	var ids []string
	for _, target := range append(append(doc.Targets("debian"), doc.Targets("ubuntu")...), doc.Targets("fedora")...) {
		ids = append(ids, target.ID)
	}
	for _, want := range []string{
		"debian-bullseye-amd64-netboot",
		"debian-bookworm-amd64-netboot",
		"debian-forky-amd64-netboot",
		"debian-trixie-amd64-netboot",
		"ubuntu-24044-amd64-netboot",
		"ubuntu-2510-amd64-netboot",
		"ubuntu-2604-amd64-netboot",
		"fedora-43-amd64-server-netboot",
		"fedora-44-amd64-server-netboot",
	} {
		if !slices.Contains(ids, want) {
			t.Fatalf("default catalog IDs = %v, want %s", ids, want)
		}
	}
	ubuntuTargets := doc.Targets("ubuntu")
	for _, target := range ubuntuTargets {
		if target.ID == "ubuntu-24044-amd64-netboot" {
			if target.Source.BaseURL != "https://releases.ubuntu.com/24.04" {
				t.Fatalf("Ubuntu 24.04.4 source base URL = %q", target.Source.BaseURL)
			}
			if target.Source.ISOName != "ubuntu-24.04.4-live-server-amd64.iso" {
				t.Fatalf("Ubuntu 24.04.4 ISO name = %q", target.Source.ISOName)
			}
			return
		}
	}
	t.Fatalf("default Ubuntu targets = %#v, want 24.04.4 sourceful target", ubuntuTargets)
}

func TestGeneratedDefaultCatalogIsCurrent(t *testing.T) {
	t.Parallel()

	generated, err := catalog.GenerateDefault()
	if err != nil {
		t.Fatalf("generate default catalog: %v", err)
	}
	current, err := os.ReadFile("default.json")
	if err != nil {
		t.Fatalf("read default catalog: %v", err)
	}
	if !bytes.Equal(generated, current) {
		t.Fatal("internal/catalog/default.json is stale; run go generate ./internal/catalog")
	}
}

func TestGeneratedDefaultPreservesFedoraLifecycle(t *testing.T) {
	t.Parallel()

	doc, err := catalog.LoadDefault([]string{"debian", "ubuntu", "fedora"})
	if err != nil {
		t.Fatalf("load default catalog: %v", err)
	}
	for _, target := range doc.Targets("fedora") {
		if target.ID != "fedora-44-amd64-server-netboot" {
			continue
		}
		if target.Source.BaseURL != "https://download.fedoraproject.org/pub/fedora/linux/releases/44/Server/x86_64/os" {
			t.Fatalf("Fedora 44 source base URL = %q", target.Source.BaseURL)
		}
		if target.Lifecycle.Status != "supported" || target.Lifecycle.Source != "catalog" {
			t.Fatalf("Fedora 44 lifecycle = %#v, want catalog supported", target.Lifecycle)
		}
		return
	}
	t.Fatalf("default Fedora targets = %#v, want Fedora 44 target", doc.Targets("fedora"))
}

func TestLoadFileLoadsLocalCatalog(t *testing.T) {
	t.Parallel()

	path := t.TempDir() + "/catalog.json"
	data := `{
		"schema_version": 1,
		"targets": [{
			"id": "debian-trixie-amd64-netboot",
			"provider_id": "debian",
			"name": "Debian trixie amd64 netboot",
			"catalog": {
				"distribution": "debian",
				"release": "trixie",
				"architecture": "amd64",
				"kind": "installer"
			}
		}]
	}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write catalog: %v", err)
	}

	doc, err := catalog.LoadFile(path, []string{"debian"})
	if err != nil {
		t.Fatalf("load file: %v", err)
	}
	if got := doc.Targets("debian")[0].ID; got != "debian-trixie-amd64-netboot" {
		t.Fatalf("loaded target ID = %q", got)
	}
}

func TestParseRejectsInvalidCatalogs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data string
	}{
		{
			name: "unsupported schema",
			data: `{"schema_version": 2, "targets": []}`,
		},
		{
			name: "missing target metadata",
			data: `{"schema_version": 1, "targets": [{
				"id": "debian-trixie-amd64-netboot",
				"provider_id": "debian",
				"name": "Debian trixie amd64 netboot",
				"catalog": {"distribution": "debian", "release": "trixie", "architecture": "amd64"}
			}]}`,
		},
		{
			name: "duplicate target id",
			data: `{"schema_version": 1, "targets": [
				{"id": "debian-trixie-amd64-netboot", "provider_id": "debian", "name": "one", "catalog": {"distribution": "debian", "release": "trixie", "architecture": "amd64", "kind": "installer"}},
				{"id": "debian-trixie-amd64-netboot", "provider_id": "debian", "name": "two", "catalog": {"distribution": "debian", "release": "trixie", "architecture": "amd64", "kind": "installer"}}
			]}`,
		},
		{
			name: "unknown provider",
			data: `{"schema_version": 1, "targets": [{
				"id": "fedora-rawhide-amd64-netboot",
				"provider_id": "fedora",
				"name": "Fedora Rawhide amd64 netboot",
				"catalog": {"distribution": "fedora", "release": "rawhide", "architecture": "amd64", "kind": "installer"}
			}]}`,
		},
		{
			name: "invalid source base url",
			data: `{"schema_version": 1, "targets": [{
				"id": "ubuntu-2604-amd64-netboot",
				"provider_id": "ubuntu",
				"name": "Ubuntu 26.04 amd64 netboot",
				"catalog": {"distribution": "ubuntu", "release": "26.04", "architecture": "amd64", "kind": "installer"},
				"source": {"base_url": "file:///srv/releases/26.04"}
			}]}`,
		},
		{
			name: "invalid source iso name",
			data: `{"schema_version": 1, "targets": [{
				"id": "ubuntu-2604-amd64-netboot",
				"provider_id": "ubuntu",
				"name": "Ubuntu 26.04 amd64 netboot",
				"catalog": {"distribution": "ubuntu", "release": "26.04", "architecture": "amd64", "kind": "installer"},
				"source": {"iso_name": "../ubuntu.iso"}
			}]}`,
		},
		{
			name: "unknown field",
			data: `{"schema_version": 1, "extra": true, "targets": []}`,
		},
		{
			name: "malformed json",
			data: `{"schema_version": 1,`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := catalog.Parse([]byte(tt.data), []string{"debian", "ubuntu"})
			if !errors.Is(err, catalog.ErrInvalidCatalog) {
				t.Fatalf("parse error = %v, want %v", err, catalog.ErrInvalidCatalog)
			}
			if !strings.Contains(err.Error(), "catalog") {
				t.Fatalf("parse error = %q, want catalog context", err)
			}
		})
	}
}
