package catalog_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

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

	doc, err := catalog.LoadDefault([]string{"debian", "ubuntu", "fedora", "linux", "local", "mfsbsd"})
	if err != nil {
		t.Fatalf("load default catalog: %v", err)
	}

	var ids []string
	for _, providerID := range []string{"debian", "ubuntu", "fedora", "linux", "local", "mfsbsd"} {
		for _, target := range doc.Targets(providerID) {
			ids = append(ids, target.ID)
		}
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
		"local-disk-auto",
		"mfsbsd-142-amd64",
		"opensuse-leap-160-amd64-netboot",
		"archlinux-latest-amd64-netboot",
		"gparted-live-1813-amd64",
	} {
		if !slices.Contains(ids, want) {
			t.Fatalf("default catalog IDs = %v, want %s", ids, want)
		}
	}
	if slices.Contains(ids, "memtest86plus-800-amd64") {
		t.Fatalf("default catalog IDs = %v, want MemTest86+ excluded until it has a compatible handoff", ids)
	}
	mfsBSDTargets := doc.Targets("mfsbsd")
	if len(mfsBSDTargets) != 1 {
		t.Fatalf("mfsBSD targets = %#v, want one target", mfsBSDTargets)
	}
	if mfsBSDTargets[0].Action != "freebsd-kboot" {
		t.Fatalf("mfsBSD action = %q, want freebsd-kboot", mfsBSDTargets[0].Action)
	}
	if mfsBSDTargets[0].Source.ISOSHA256 == "" {
		t.Fatal("mfsBSD ISO SHA256 is empty")
	}
	if len(mfsBSDTargets[0].Options) != 1 {
		t.Fatalf("mfsBSD options = %#v, want hostname option", mfsBSDTargets[0].Options)
	}
	if option := mfsBSDTargets[0].Options[0]; option.ID != "hostname" || option.Template != "mfsbsd.hostname={value}" {
		t.Fatalf("mfsBSD option = %#v, want hostname template option", option)
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

func TestLoadDefaultIncludesGenericLinuxSourceTargets(t *testing.T) {
	t.Parallel()

	doc, err := catalog.LoadDefault([]string{"debian", "ubuntu", "fedora", "linux", "local", "mfsbsd"})
	if err != nil {
		t.Fatalf("load default catalog: %v", err)
	}

	linuxTargets := doc.Targets("linux")
	for _, target := range linuxTargets {
		if target.ID != "opensuse-leap-160-amd64-netboot" {
			continue
		}
		if target.Catalog.Distribution != "opensuse" {
			t.Fatalf("openSUSE distribution = %q", target.Catalog.Distribution)
		}
		if target.Source.KernelPath != "boot/x86_64/loader/linux" {
			t.Fatalf("openSUSE kernel path = %q", target.Source.KernelPath)
		}
		if target.Source.InitrdPath != "boot/x86_64/loader/initrd" {
			t.Fatalf("openSUSE initrd path = %q", target.Source.InitrdPath)
		}
		if target.Source.Cmdline == "" {
			t.Fatal("openSUSE cmdline is empty")
		}
		return
	}
	t.Fatalf("default Linux targets = %#v, want openSUSE target", linuxTargets)
}

func TestParsePreservesSourceArtifactHashPins(t *testing.T) {
	t.Parallel()

	kernelHash := strings.Repeat("a", 64)
	initrdHash := strings.Repeat("b", 64)
	doc, err := catalog.Parse([]byte(`{
		"schema_version": 1,
		"targets": [{
			"id": "opensuse-leap-160-amd64-netboot",
			"provider_id": "linux",
			"name": "openSUSE Leap 16.0 amd64 installer",
			"catalog": {
				"distribution": "opensuse",
				"release": "leap-16.0",
				"architecture": "amd64",
				"kind": "installer"
			},
			"source": {
				"base_url": "https://download.example/opensuse",
				"kernel_path": "boot/x86_64/loader/linux",
				"initrd_path": "boot/x86_64/loader/initrd",
				"kernel_sha256": "`+kernelHash+`",
				"initrd_sha256": "`+initrdHash+`"
			}
		}]
	}`), []string{"linux"})
	if err != nil {
		t.Fatalf("parse catalog: %v", err)
	}

	target := doc.Targets("linux")[0]
	if target.Source.KernelSHA256 != kernelHash {
		t.Fatalf("kernel sha256 = %q, want %q", target.Source.KernelSHA256, kernelHash)
	}
	if target.Source.InitrdSHA256 != initrdHash {
		t.Fatalf("initrd sha256 = %q, want %q", target.Source.InitrdSHA256, initrdHash)
	}
}

func TestLoadDefaultIncludesLocalBootAction(t *testing.T) {
	t.Parallel()

	doc, err := catalog.LoadDefault([]string{"debian", "ubuntu", "fedora", "linux", "local", "mfsbsd"})
	if err != nil {
		t.Fatalf("load default catalog: %v", err)
	}

	targets := doc.Targets("local")
	if len(targets) != 1 {
		t.Fatalf("local targets = %#v, want one target", targets)
	}
	if targets[0].Action != "localboot" {
		t.Fatalf("local action = %q, want localboot", targets[0].Action)
	}
}

func TestGenerateAllowsTargetDistributionDifferentFromProvider(t *testing.T) {
	t.Parallel()

	generated, err := catalog.Generate([]byte(`{
		"schema_version": 1,
		"providers": [{
			"id": "linux",
			"targets": [{
				"id": "opensuse-leap-160-amd64-netboot",
				"name": "openSUSE Leap 16.0 amd64 installer",
				"distribution": "opensuse",
				"release": "leap-16.0",
				"architecture": "amd64",
				"kind": "installer",
				"source": {
					"base_url": "https://download.example/opensuse",
					"kernel_path": "boot/x86_64/loader/linux",
					"initrd_path": "boot/x86_64/loader/initrd",
					"cmdline": "install={base_url}"
				}
			}]
		}]
	}`))
	if err != nil {
		t.Fatalf("generate catalog: %v", err)
	}
	doc, err := catalog.Parse(generated, []string{"linux"})
	if err != nil {
		t.Fatalf("parse generated catalog: %v", err)
	}
	target := doc.Targets("linux")[0]
	if target.ProviderID != "linux" || target.Catalog.Distribution != "opensuse" {
		t.Fatalf("target provider/distribution = %q/%q", target.ProviderID, target.Catalog.Distribution)
	}
}

func TestGeneratePreservesSourceArtifactHashPins(t *testing.T) {
	t.Parallel()

	kernelHash := strings.Repeat("a", 64)
	initrdHash := strings.Repeat("b", 64)
	generated, err := catalog.Generate([]byte(`{
		"schema_version": 1,
		"providers": [{
			"id": "linux",
			"targets": [{
				"id": "opensuse-leap-160-amd64-netboot",
				"name": "openSUSE Leap 16.0 amd64 installer",
				"distribution": "opensuse",
				"release": "leap-16.0",
				"architecture": "amd64",
				"kind": "installer",
				"source": {
					"base_url": "https://download.example/opensuse",
					"kernel_path": "boot/x86_64/loader/linux",
					"initrd_path": "boot/x86_64/loader/initrd",
					"kernel_sha256": "` + kernelHash + `",
					"initrd_sha256": "` + initrdHash + `",
					"cmdline": "install={base_url}"
				}
			}]
		}]
	}`))
	if err != nil {
		t.Fatalf("generate catalog: %v", err)
	}

	doc, err := catalog.Parse(generated, []string{"linux"})
	if err != nil {
		t.Fatalf("parse generated catalog: %v", err)
	}
	target := doc.Targets("linux")[0]
	if target.Source.KernelSHA256 != kernelHash {
		t.Fatalf("kernel sha256 = %q, want %q", target.Source.KernelSHA256, kernelHash)
	}
	if target.Source.InitrdSHA256 != initrdHash {
		t.Fatalf("initrd sha256 = %q, want %q", target.Source.InitrdSHA256, initrdHash)
	}
}

func TestGeneratePreservesTargetOptions(t *testing.T) {
	t.Parallel()

	generated, err := catalog.Generate([]byte(`{
		"schema_version": 1,
		"providers": [{
			"id": "linux",
			"targets": [{
				"id": "opensuse-leap-160-amd64-netboot",
				"name": "openSUSE Leap 16.0 amd64 installer",
				"distribution": "opensuse",
				"release": "leap-16.0",
				"architecture": "amd64",
				"kind": "installer",
				"source": {
					"base_url": "https://download.example/opensuse",
					"kernel_path": "boot/x86_64/loader/linux",
					"initrd_path": "boot/x86_64/loader/initrd",
					"cmdline": "install={base_url}"
				},
				"options": [
					{
						"id": "serial",
						"label": "Serial console",
						"type": "bool",
						"fragment": "console=ttyS0"
					},
					{
						"id": "install-mode",
						"label": "Install mode",
						"type": "enum",
						"values": [
							{"value": "text", "label": "Text install", "fragment": "textmode=1"},
							{"value": "vnc", "label": "VNC install", "fragment": "vnc=1"}
						]
					},
					{
						"id": "mirror",
						"label": "Mirror URL",
						"type": "string",
						"template": "install={value}"
					}
				]
			}]
		}]
	}`))
	if err != nil {
		t.Fatalf("generate catalog: %v", err)
	}
	doc, err := catalog.Parse(generated, []string{"linux"})
	if err != nil {
		t.Fatalf("parse generated catalog: %v", err)
	}
	target := doc.Targets("linux")[0]
	if len(target.Options) != 3 {
		t.Fatalf("target options = %#v, want three options", target.Options)
	}
	if target.Options[0].ID != "serial" || target.Options[0].Fragment != "console=ttyS0" {
		t.Fatalf("first option = %#v, want serial console bool option", target.Options[0])
	}
	if got := target.Options[1].Values[1].Fragment; got != "vnc=1" {
		t.Fatalf("enum value fragment = %q, want vnc=1", got)
	}
	if got := target.Options[2].Template; got != "install={value}" {
		t.Fatalf("string option template = %q, want install={value}", got)
	}
}

func TestGeneratePreservesTargetSecretDeclarations(t *testing.T) {
	t.Parallel()

	generated, err := catalog.Generate([]byte(`{
		"schema_version": 1,
		"providers": [{
			"id": "linux",
			"targets": [{
				"id": "site-installer",
				"name": "Site installer",
				"distribution": "site",
				"release": "current",
				"architecture": "amd64",
				"kind": "installer",
				"source": {
					"base_url": "https://download.example/site",
					"kernel_path": "linux",
					"cmdline": "console=ttyS0"
				},
				"secrets": [{
					"id": "installer-password",
					"label": "Installer password",
					"purpose": "Used by the installer automation profile.",
					"required": true,
					"delivery": "staged-file"
				}]
			}]
		}]
	}`))
	if err != nil {
		t.Fatalf("generate catalog: %v", err)
	}
	doc, err := catalog.Parse(generated, []string{"linux"})
	if err != nil {
		t.Fatalf("parse generated catalog: %v", err)
	}
	target := doc.Targets("linux")[0]
	if len(target.Secrets) != 1 {
		t.Fatalf("target secrets = %#v, want one secret", target.Secrets)
	}
	secret := target.Secrets[0]
	if secret.ID != "installer-password" || !secret.Required || string(secret.Delivery) != "staged-file" {
		t.Fatalf("secret declaration = %#v, want required staged-file installer-password", secret)
	}
	if secret.Label == "" || secret.Purpose == "" {
		t.Fatalf("secret declaration = %#v, want label and purpose", secret)
	}
}

func TestGenerateRejectsInvalidTargetOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		options string
	}{
		{
			name:    "missing id",
			options: `[{"label": "Serial console", "type": "bool", "fragment": "console=ttyS0"}]`,
		},
		{
			name: "duplicate ids",
			options: `[
				{"id": "serial", "label": "Serial console", "type": "bool", "fragment": "console=ttyS0"},
				{"id": "serial", "label": "Serial console again", "type": "bool", "fragment": "console=ttyS1"}
			]`,
		},
		{
			name:    "unsupported type",
			options: `[{"id": "serial", "label": "Serial console", "type": "script", "fragment": "console=ttyS0"}]`,
		},
		{
			name:    "missing enum values",
			options: `[{"id": "install-mode", "label": "Install mode", "type": "enum"}]`,
		},
		{
			name: "duplicate enum values",
			options: `[{
				"id": "install-mode",
				"label": "Install mode",
				"type": "enum",
				"values": [
					{"value": "text", "fragment": "textmode=1"},
					{"value": "text", "fragment": "textmode=1"}
				]
			}]`,
		},
		{
			name: "invalid enum value",
			options: `[{
				"id": "install-mode",
				"label": "Install mode",
				"type": "enum",
				"values": [
					{"value": "bad value", "fragment": "textmode=1"}
				]
			}]`,
		},
		{
			name:    "malformed fragment",
			options: `[{"id": "serial", "label": "Serial console", "type": "bool", "fragment": " console=ttyS0"}]`,
		},
		{
			name:    "string template without value",
			options: `[{"id": "mirror", "label": "Mirror URL", "type": "string", "template": "install=https://example.test"}]`,
		},
		{
			name:    "executable behavior",
			options: `[{"id": "hook", "label": "Runtime hook", "type": "bool", "fragment": "hook=1", "script": "echo unsafe"}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data := strings.ReplaceAll(`{
				"schema_version": 1,
				"providers": [{
					"id": "linux",
					"targets": [{
						"id": "opensuse-leap-160-amd64-netboot",
						"name": "openSUSE Leap 16.0 amd64 installer",
						"distribution": "opensuse",
						"release": "leap-16.0",
						"architecture": "amd64",
						"kind": "installer",
						"source": {
							"base_url": "https://download.example/opensuse",
							"kernel_path": "boot/x86_64/loader/linux",
							"initrd_path": "boot/x86_64/loader/initrd",
							"cmdline": "install={base_url}"
						},
						"options": __OPTIONS__
					}]
				}]
			}`, "__OPTIONS__", tt.options)
			_, err := catalog.Generate([]byte(data))
			if !errors.Is(err, catalog.ErrInvalidCatalog) {
				t.Fatalf("generate error = %v, want %v", err, catalog.ErrInvalidCatalog)
			}
		})
	}
}

func TestGenerateRejectsInvalidSecretDeclarations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		secrets string
	}{
		{
			name:    "missing purpose",
			secrets: `[{"id": "installer-password", "label": "Installer password", "required": true, "delivery": "staged-file"}]`,
		},
		{
			name: "duplicate ids",
			secrets: `[
				{"id": "installer-password", "label": "Installer password", "purpose": "Used by automation.", "required": true, "delivery": "staged-file"},
				{"id": "installer-password", "label": "Installer password again", "purpose": "Used by automation.", "required": true, "delivery": "staged-file"}
			]`,
		},
		{
			name:    "unsupported delivery",
			secrets: `[{"id": "installer-password", "label": "Installer password", "purpose": "Used by automation.", "required": true, "delivery": "inline"}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data := strings.ReplaceAll(`{
				"schema_version": 1,
				"providers": [{
					"id": "linux",
					"targets": [{
						"id": "site-installer",
						"name": "Site installer",
						"distribution": "site",
						"release": "current",
						"architecture": "amd64",
						"kind": "installer",
						"source": {
							"base_url": "https://download.example/site",
							"kernel_path": "linux",
							"cmdline": "console=ttyS0"
						},
						"secrets": __SECRETS__
					}]
				}]
			}`, "__SECRETS__", tt.secrets)
			_, err := catalog.Generate([]byte(data))
			if !errors.Is(err, catalog.ErrInvalidCatalog) {
				t.Fatalf("generate error = %v, want %v", err, catalog.ErrInvalidCatalog)
			}
		})
	}
}

func TestGenerateRejectsSecretTargetOptionWithExplicitError(t *testing.T) {
	t.Parallel()

	data := []byte(`{
		"schema_version": 1,
		"providers": [{
			"id": "mfsbsd",
			"targets": [{
				"id": "mfsbsd-142-amd64",
				"name": "mfsBSD 14.2 amd64",
				"release": "14.2",
				"architecture": "amd64",
				"kind": "rescue",
				"options": [{
					"id": "root-password",
					"label": "Root password",
					"type": "string",
					"template": "mfsbsd.root_password={value}",
					"secret": true
				}]
			}]
		}]
	}`)

	_, err := catalog.Generate(data)
	if !errors.Is(err, catalog.ErrInvalidCatalog) {
		t.Fatalf("generate error = %v, want %v", err, catalog.ErrInvalidCatalog)
	}
	if !strings.Contains(err.Error(), "secret target options are not supported") {
		t.Fatalf("generate error = %q, want secret option boundary", err)
	}
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

	doc, err := catalog.LoadDefault([]string{"debian", "ubuntu", "fedora", "linux", "local", "mfsbsd"})
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

func TestComposeAppendsCatalogTargets(t *testing.T) {
	t.Parallel()

	base := parseTestCatalog(t, `{
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
	}`, []string{"debian", "linux"})
	extra := parseTestCatalog(t, `{
		"schema_version": 1,
		"targets": [{
			"id": "opensuse-lab-amd64-netboot",
			"provider_id": "linux",
			"name": "openSUSE lab amd64 netboot",
			"catalog": {
				"distribution": "opensuse",
				"release": "lab",
				"architecture": "amd64",
				"kind": "installer"
			},
			"source": {
				"base_url": "https://download.example/opensuse",
				"kernel_path": "boot/x86_64/loader/linux",
				"kernel_sha256": "`+strings.Repeat("a", 64)+`"
			}
		}]
	}`, []string{"debian", "linux"})

	doc, err := catalog.Compose(base, extra)
	if err != nil {
		t.Fatalf("compose catalogs: %v", err)
	}
	if len(doc.Entries) != 2 {
		t.Fatalf("entries length = %d, want 2", len(doc.Entries))
	}
	if doc.Entries[0].ID != "debian-trixie-amd64-netboot" || doc.Entries[1].ID != "opensuse-lab-amd64-netboot" {
		t.Fatalf("entries = %#v, want base target then extra target", doc.Entries)
	}
}

func TestComposeRejectsDuplicateTargetID(t *testing.T) {
	t.Parallel()

	first := parseTestCatalog(t, `{
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
	}`, []string{"debian"})
	second := parseTestCatalog(t, `{
		"schema_version": 1,
		"targets": [{
			"id": "debian-trixie-amd64-netboot",
			"provider_id": "debian",
			"name": "Duplicate Debian trixie",
			"catalog": {
				"distribution": "debian",
				"release": "trixie",
				"architecture": "amd64",
				"kind": "installer"
			}
		}]
	}`, []string{"debian"})

	_, err := catalog.Compose(first, second)
	if err == nil {
		t.Fatal("compose succeeded, want duplicate target error")
	}
	if !strings.Contains(err.Error(), "duplicate target ID") {
		t.Fatalf("compose error = %q, want duplicate target context", err)
	}
}

func TestParseCatalogFreshnessMetadata(t *testing.T) {
	t.Parallel()

	doc, err := catalog.Parse([]byte(`{
		"schema_version": 1,
		"published_at": "2026-05-07T08:00:00Z",
		"expires_at": "2026-05-08T08:00:00Z",
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
	}`), []string{"debian"})
	if err != nil {
		t.Fatalf("parse catalog: %v", err)
	}
	if doc.PublishedAt == nil || !doc.PublishedAt.Equal(time.Date(2026, 5, 7, 8, 0, 0, 0, time.UTC)) {
		t.Fatalf("published_at = %v, want 2026-05-07T08:00:00Z", doc.PublishedAt)
	}
	if doc.ExpiresAt == nil || !doc.ExpiresAt.Equal(time.Date(2026, 5, 8, 8, 0, 0, 0, time.UTC)) {
		t.Fatalf("expires_at = %v, want 2026-05-08T08:00:00Z", doc.ExpiresAt)
	}
}

func parseTestCatalog(t *testing.T, data string, providerIDs []string) catalog.Document {
	t.Helper()

	doc, err := catalog.Parse([]byte(data), providerIDs)
	if err != nil {
		t.Fatalf("parse catalog: %v", err)
	}
	return doc
}

func TestVerifyHostedTrustAcceptsSHA256Pin(t *testing.T) {
	t.Parallel()

	data := []byte(`{"schema_version":1,"targets":[]}`)
	sum := sha256.Sum256(data)
	if err := catalog.VerifyHostedTrust(data, catalog.HostedTrust{
		SHA256: hex.EncodeToString(sum[:]),
	}); err != nil {
		t.Fatalf("verify trust: %v", err)
	}
}

func TestVerifyHostedTrustRejectsSHA256Mismatch(t *testing.T) {
	t.Parallel()

	err := catalog.VerifyHostedTrust([]byte(`{"schema_version":1,"targets":[]}`), catalog.HostedTrust{
		SHA256: strings.Repeat("0", sha256.Size*2),
	})
	if !errors.Is(err, catalog.ErrInvalidCatalog) {
		t.Fatalf("verify trust error = %v, want %v", err, catalog.ErrInvalidCatalog)
	}
	if !strings.Contains(err.Error(), "SHA-256") {
		t.Fatalf("verify trust error = %q, want SHA-256 context", err)
	}
}

func TestVerifyHostedTrustAcceptsEd25519Signature(t *testing.T) {
	t.Parallel()

	data := []byte(`{"schema_version":1,"targets":[]}`)
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	signature := ed25519.Sign(privateKey, data)

	if err := catalog.VerifyHostedTrust(data, catalog.HostedTrust{
		Ed25519Signature: signature,
		Ed25519PublicKey: publicKey,
	}); err != nil {
		t.Fatalf("verify trust: %v", err)
	}
}

func TestVerifyHostedTrustRejectsEd25519Mismatch(t *testing.T) {
	t.Parallel()

	data := []byte(`{"schema_version":1,"targets":[]}`)
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	signature := ed25519.Sign(privateKey, []byte("different"))

	err = catalog.VerifyHostedTrust(data, catalog.HostedTrust{
		Ed25519Signature: signature,
		Ed25519PublicKey: publicKey,
	})
	if !errors.Is(err, catalog.ErrInvalidCatalog) {
		t.Fatalf("verify trust error = %v, want %v", err, catalog.ErrInvalidCatalog)
	}
	if !strings.Contains(err.Error(), "Ed25519") {
		t.Fatalf("verify trust error = %q, want Ed25519 context", err)
	}
}

func TestVerifyHostedTrustRequiresTrustConfiguration(t *testing.T) {
	t.Parallel()

	err := catalog.VerifyHostedTrust([]byte(`{"schema_version":1,"targets":[]}`), catalog.HostedTrust{})
	if !errors.Is(err, catalog.ErrInvalidCatalog) {
		t.Fatalf("verify trust error = %v, want %v", err, catalog.ErrInvalidCatalog)
	}
	if !strings.Contains(err.Error(), "trust") {
		t.Fatalf("verify trust error = %q, want trust context", err)
	}
}

func TestLoadHostedCatalogFetchesAuthenticatesAndParsesHTTPSCatalog(t *testing.T) {
	t.Parallel()

	data := []byte(validHostedCatalogJSON)
	sum := sha256.Sum256(data)
	client := &http.Client{Transport: catalogResponseMap{
		"https://catalog.example/bootup/catalog.json": catalogResponse{
			statusCode: http.StatusOK,
			body:       data,
		},
	}}

	doc, err := catalog.LoadHosted(context.Background(), catalog.HostedOptions{
		URL:         "https://catalog.example/bootup/catalog.json",
		HTTPClient:  client,
		ProviderIDs: []string{"debian"},
		Trust: catalog.HostedTrust{
			SHA256: hex.EncodeToString(sum[:]),
		},
	})
	if err != nil {
		t.Fatalf("load hosted catalog: %v", err)
	}
	if got := doc.Targets("debian")[0].ID; got != "debian-trixie-amd64-netboot" {
		t.Fatalf("hosted target = %q, want Debian trixie", got)
	}
}

func TestLoadHostedCatalogRejectsUnsupportedScheme(t *testing.T) {
	t.Parallel()

	_, err := catalog.LoadHosted(context.Background(), catalog.HostedOptions{
		URL:         "http://catalog.example/bootup/catalog.json",
		ProviderIDs: []string{"debian"},
		Trust: catalog.HostedTrust{
			SHA256: strings.Repeat("0", sha256.Size*2),
		},
	})
	if !errors.Is(err, catalog.ErrInvalidCatalog) {
		t.Fatalf("load hosted error = %v, want %v", err, catalog.ErrInvalidCatalog)
	}
	if !strings.Contains(err.Error(), "https") {
		t.Fatalf("load hosted error = %q, want HTTPS context", err)
	}
}

func TestLoadHostedCatalogRejectsHTTPError(t *testing.T) {
	t.Parallel()

	client := &http.Client{Transport: catalogResponseMap{
		"https://catalog.example/bootup/catalog.json": catalogResponse{
			statusCode: http.StatusInternalServerError,
			body:       []byte("error"),
		},
	}}

	_, err := catalog.LoadHosted(context.Background(), catalog.HostedOptions{
		URL:         "https://catalog.example/bootup/catalog.json",
		HTTPClient:  client,
		ProviderIDs: []string{"debian"},
		Trust: catalog.HostedTrust{
			SHA256: strings.Repeat("0", sha256.Size*2),
		},
	})
	if err == nil {
		t.Fatal("load hosted succeeded, want HTTP status error")
	}
	if !strings.Contains(err.Error(), "Internal Server Error") {
		t.Fatalf("load hosted error = %q, want status text", err)
	}
}

func TestLoadHostedCatalogRejectsOversizedResponse(t *testing.T) {
	t.Parallel()

	client := &http.Client{Transport: catalogResponseMap{
		"https://catalog.example/bootup/catalog.json": catalogResponse{
			statusCode: http.StatusOK,
			body:       []byte(validHostedCatalogJSON),
		},
	}}

	_, err := catalog.LoadHosted(context.Background(), catalog.HostedOptions{
		URL:         "https://catalog.example/bootup/catalog.json",
		HTTPClient:  client,
		ProviderIDs: []string{"debian"},
		MaxBytes:    16,
		Trust: catalog.HostedTrust{
			SHA256: strings.Repeat("0", sha256.Size*2),
		},
	})
	if err == nil {
		t.Fatal("load hosted succeeded, want oversized response error")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Fatalf("load hosted error = %q, want size context", err)
	}
}

func TestLoadHostedCatalogRejectsExpiredCatalog(t *testing.T) {
	t.Parallel()

	data := []byte(`{
		"schema_version": 1,
		"expires_at": "2026-05-07T08:00:00Z",
		"targets": []
	}`)
	client := &http.Client{Transport: catalogResponseMap{
		"https://catalog.example/bootup/catalog.json": catalogResponse{
			statusCode: http.StatusOK,
			body:       data,
		},
	}}

	_, err := catalog.LoadHosted(context.Background(), catalog.HostedOptions{
		URL:         "https://catalog.example/bootup/catalog.json",
		HTTPClient:  client,
		ProviderIDs: []string{"debian"},
		Trust:       digestTrust(data),
		Now:         fixedNow(2026, 5, 7, 9, 0),
	})
	if !errors.Is(err, catalog.ErrInvalidCatalog) {
		t.Fatalf("load hosted error = %v, want %v", err, catalog.ErrInvalidCatalog)
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Fatalf("load hosted error = %q, want expiry context", err)
	}
}

func TestLoadHostedCatalogRejectsCatalogPastMaximumAge(t *testing.T) {
	t.Parallel()

	data := []byte(`{
		"schema_version": 1,
		"published_at": "2026-05-07T08:00:00Z",
		"targets": []
	}`)
	client := &http.Client{Transport: catalogResponseMap{
		"https://catalog.example/bootup/catalog.json": catalogResponse{
			statusCode: http.StatusOK,
			body:       data,
		},
	}}

	_, err := catalog.LoadHosted(context.Background(), catalog.HostedOptions{
		URL:         "https://catalog.example/bootup/catalog.json",
		HTTPClient:  client,
		ProviderIDs: []string{"debian"},
		Trust:       digestTrust(data),
		Now:         fixedNow(2026, 5, 7, 10, 0),
		MaxAge:      time.Hour,
	})
	if !errors.Is(err, catalog.ErrInvalidCatalog) {
		t.Fatalf("load hosted error = %v, want %v", err, catalog.ErrInvalidCatalog)
	}
	if !strings.Contains(err.Error(), "maximum age") {
		t.Fatalf("load hosted error = %q, want maximum age context", err)
	}
}

func TestLoadHostedCatalogRejectsMissingRequiredFreshness(t *testing.T) {
	t.Parallel()

	data := []byte(`{"schema_version": 1, "targets": []}`)
	client := &http.Client{Transport: catalogResponseMap{
		"https://catalog.example/bootup/catalog.json": catalogResponse{
			statusCode: http.StatusOK,
			body:       data,
		},
	}}

	_, err := catalog.LoadHosted(context.Background(), catalog.HostedOptions{
		URL:              "https://catalog.example/bootup/catalog.json",
		HTTPClient:       client,
		ProviderIDs:      []string{"debian"},
		Trust:            digestTrust(data),
		RequireFreshness: true,
	})
	if !errors.Is(err, catalog.ErrInvalidCatalog) {
		t.Fatalf("load hosted error = %v, want %v", err, catalog.ErrInvalidCatalog)
	}
	if !strings.Contains(err.Error(), "freshness") {
		t.Fatalf("load hosted error = %q, want freshness context", err)
	}
}

func TestLoadHostedCatalogWritesAuthenticatedCache(t *testing.T) {
	t.Parallel()

	data := []byte(validHostedCatalogJSON)
	cachePath := t.TempDir() + "/catalog-cache.json"
	client := &http.Client{Transport: catalogResponseMap{
		"https://catalog.example/bootup/catalog.json": catalogResponse{
			statusCode: http.StatusOK,
			body:       data,
		},
	}}

	_, err := catalog.LoadHosted(context.Background(), catalog.HostedOptions{
		URL:         "https://catalog.example/bootup/catalog.json",
		HTTPClient:  client,
		ProviderIDs: []string{"debian"},
		Trust:       digestTrust(data),
		CachePath:   cachePath,
	})
	if err != nil {
		t.Fatalf("load hosted catalog: %v", err)
	}
	cached, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("read cache: %v", err)
	}
	if !bytes.Equal(cached, data) {
		t.Fatalf("cache bytes = %q, want hosted catalog", cached)
	}
}

func TestLoadHostedCatalogFallsBackToAuthenticatedCache(t *testing.T) {
	t.Parallel()

	data := []byte(validHostedCatalogJSON)
	cachePath := t.TempDir() + "/catalog-cache.json"
	if err := os.WriteFile(cachePath, data, 0o644); err != nil {
		t.Fatalf("write cache: %v", err)
	}
	client := &http.Client{Transport: catalogResponseMap{
		"https://catalog.example/bootup/catalog.json": catalogResponse{
			statusCode: http.StatusInternalServerError,
			body:       []byte("error"),
		},
	}}

	doc, err := catalog.LoadHosted(context.Background(), catalog.HostedOptions{
		URL:           "https://catalog.example/bootup/catalog.json",
		HTTPClient:    client,
		ProviderIDs:   []string{"debian"},
		Trust:         digestTrust(data),
		CachePath:     cachePath,
		CacheFallback: true,
	})
	if err != nil {
		t.Fatalf("load hosted catalog: %v", err)
	}
	if got := doc.Targets("debian")[0].ID; got != "debian-trixie-amd64-netboot" {
		t.Fatalf("cached hosted target = %q, want Debian trixie", got)
	}
}

func TestLoadHostedCatalogRejectsStaleCacheFallback(t *testing.T) {
	t.Parallel()

	data := []byte(`{
		"schema_version": 1,
		"expires_at": "2026-05-07T08:00:00Z",
		"targets": []
	}`)
	cachePath := t.TempDir() + "/catalog-cache.json"
	if err := os.WriteFile(cachePath, data, 0o644); err != nil {
		t.Fatalf("write cache: %v", err)
	}
	client := &http.Client{Transport: catalogResponseMap{
		"https://catalog.example/bootup/catalog.json": catalogResponse{
			statusCode: http.StatusInternalServerError,
			body:       []byte("error"),
		},
	}}

	_, err := catalog.LoadHosted(context.Background(), catalog.HostedOptions{
		URL:           "https://catalog.example/bootup/catalog.json",
		HTTPClient:    client,
		ProviderIDs:   []string{"debian"},
		Trust:         digestTrust(data),
		CachePath:     cachePath,
		CacheFallback: true,
		Now:           fixedNow(2026, 5, 7, 9, 0),
	})
	if !errors.Is(err, catalog.ErrInvalidCatalog) {
		t.Fatalf("load hosted error = %v, want %v", err, catalog.ErrInvalidCatalog)
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Fatalf("load hosted error = %q, want expiry context", err)
	}
}

func TestLoadHostedCatalogRejectsUnauthenticatedCacheFallback(t *testing.T) {
	t.Parallel()

	data := []byte(validHostedCatalogJSON)
	cachePath := t.TempDir() + "/catalog-cache.json"
	if err := os.WriteFile(cachePath, data, 0o644); err != nil {
		t.Fatalf("write cache: %v", err)
	}
	client := &http.Client{Transport: catalogResponseMap{
		"https://catalog.example/bootup/catalog.json": catalogResponse{
			statusCode: http.StatusInternalServerError,
			body:       []byte("error"),
		},
	}}

	_, err := catalog.LoadHosted(context.Background(), catalog.HostedOptions{
		URL:           "https://catalog.example/bootup/catalog.json",
		HTTPClient:    client,
		ProviderIDs:   []string{"debian"},
		Trust:         catalog.HostedTrust{SHA256: strings.Repeat("0", sha256.Size*2)},
		CachePath:     cachePath,
		CacheFallback: true,
	})
	if !errors.Is(err, catalog.ErrInvalidCatalog) {
		t.Fatalf("load hosted error = %v, want %v", err, catalog.ErrInvalidCatalog)
	}
	if !strings.Contains(err.Error(), "SHA-256") {
		t.Fatalf("load hosted error = %q, want SHA-256 context", err)
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
			name: "invalid source kernel sha256",
			data: `{"schema_version": 1, "targets": [{
				"id": "ubuntu-2604-amd64-netboot",
				"provider_id": "ubuntu",
				"name": "Ubuntu 26.04 amd64 netboot",
				"catalog": {"distribution": "ubuntu", "release": "26.04", "architecture": "amd64", "kind": "installer"},
				"source": {"kernel_sha256": "not-a-sha256"}
			}]}`,
		},
		{
			name: "partial initrd hash pins",
			data: `{"schema_version": 1, "targets": [{
				"id": "ubuntu-2604-amd64-netboot",
				"provider_id": "ubuntu",
				"name": "Ubuntu 26.04 amd64 netboot",
				"catalog": {"distribution": "ubuntu", "release": "26.04", "architecture": "amd64", "kind": "installer"},
				"source": {"kernel_path": "netboot/linux", "initrd_path": "netboot/initrd", "kernel_sha256": "` + strings.Repeat("a", 64) + `"}
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
		{
			name: "malformed published timestamp",
			data: `{"schema_version": 1, "published_at": "not a timestamp", "targets": []}`,
		},
		{
			name: "malformed expiry timestamp",
			data: `{"schema_version": 1, "expires_at": "not a timestamp", "targets": []}`,
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

const validHostedCatalogJSON = `{
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

type catalogResponse struct {
	statusCode int
	body       []byte
}

type catalogResponseMap map[string]catalogResponse

func (m catalogResponseMap) RoundTrip(request *http.Request) (*http.Response, error) {
	item, ok := m[request.URL.String()]
	if !ok {
		item = catalogResponse{statusCode: http.StatusNotFound, body: []byte("not found")}
	}
	return &http.Response{
		StatusCode: item.statusCode,
		Status:     http.StatusText(item.statusCode),
		Body:       io.NopCloser(bytes.NewReader(item.body)),
		Header:     make(http.Header),
		Request:    request,
	}, nil
}

func digestTrust(data []byte) catalog.HostedTrust {
	sum := sha256.Sum256(data)
	return catalog.HostedTrust{SHA256: hex.EncodeToString(sum[:])}
}

func fixedNow(year int, month time.Month, day int, hour int, minute int) func() time.Time {
	return func() time.Time {
		return time.Date(year, month, day, hour, minute, 0, 0, time.UTC)
	}
}
