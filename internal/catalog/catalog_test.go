package catalog_test

import (
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
}

func TestLoadDefaultIncludesInitialStaticTargets(t *testing.T) {
	t.Parallel()

	doc, err := catalog.LoadDefault([]string{"debian", "ubuntu"})
	if err != nil {
		t.Fatalf("load default catalog: %v", err)
	}

	var ids []string
	for _, target := range append(doc.Targets("debian"), doc.Targets("ubuntu")...) {
		ids = append(ids, target.ID)
	}
	for _, want := range []string{
		"debian-bookworm-amd64-netboot",
		"debian-trixie-amd64-netboot",
		"ubuntu-2604-amd64-netboot",
	} {
		if !slices.Contains(ids, want) {
			t.Fatalf("default catalog IDs = %v, want %s", ids, want)
		}
	}
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
