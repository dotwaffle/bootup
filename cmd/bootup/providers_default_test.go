//go:build !bootup_debian_fixture

package main

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/dotwaffle/bootup/internal/catalog"
	"github.com/dotwaffle/bootup/internal/provider"
	"github.com/dotwaffle/bootup/internal/providerconfig"
)

func TestRegisterProvidersIncludesDefaultCatalogTargets(t *testing.T) {
	t.Parallel()

	catalogDoc, err := catalog.LoadDefault(compiledProviderIDs())
	if err != nil {
		t.Fatalf("load default catalog: %v", err)
	}
	registry := provider.NewRegistry()
	if err := registerProviders(registry, providerconfig.Config{}, catalogDoc); err != nil {
		t.Fatalf("register providers: %v", err)
	}
	targets, err := registry.Targets(context.Background())
	if err != nil {
		t.Fatalf("targets: %v", err)
	}
	var ids []string
	for _, target := range targets {
		ids = append(ids, target.ID)
	}
	for _, want := range []string{
		"debian-bookworm-amd64-netboot",
		"debian-trixie-amd64-netboot",
		"ubuntu-2604-amd64-netboot",
	} {
		if !slices.Contains(ids, want) {
			t.Fatalf("registered targets = %v, want %s", ids, want)
		}
	}
}

func TestRegisterProvidersUsesCatalogDocumentAsReplacement(t *testing.T) {
	t.Parallel()

	catalogDoc, err := catalog.Parse([]byte(`{
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
	}`), compiledProviderIDs())
	if err != nil {
		t.Fatalf("parse catalog: %v", err)
	}

	registry := provider.NewRegistry()
	if err := registerProviders(registry, providerconfig.Config{}, catalogDoc); err != nil {
		t.Fatalf("register providers: %v", err)
	}
	targets, err := registry.Targets(context.Background())
	if err != nil {
		t.Fatalf("targets: %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("targets length = %d, want 1: %#v", len(targets), targets)
	}
	if targets[0].ID != "debian-trixie-amd64-netboot" {
		t.Fatalf("target ID = %q, want Debian trixie", targets[0].ID)
	}
}

func TestRegisterProvidersAppliesRuntimeConfig(t *testing.T) {
	t.Parallel()

	catalogDoc, err := catalog.LoadDefault(compiledProviderIDs())
	if err != nil {
		t.Fatalf("load default catalog: %v", err)
	}
	registry := provider.NewRegistry()
	if err := registerProviders(registry, providerconfig.Config{
		Debian: providerconfig.DebianConfig{
			MirrorURL: "https://mirror.example/debian",
			Keyring:   []byte("debian keyring"),
		},
		Ubuntu: providerconfig.UbuntuConfig{
			ReleaseURL:   "https://releases.example/26.04",
			Keyring:      []byte("ubuntu keyring"),
			KernelSHA256: strings.Repeat("a", 64),
			InitrdSHA256: strings.Repeat("b", 64),
		},
	}, catalogDoc); err != nil {
		t.Fatalf("register providers: %v", err)
	}

	debianPlan, err := registry.Plan(context.Background(), provider.Target{
		ID:         "debian-trixie-amd64-netboot",
		ProviderID: "debian",
	})
	if err != nil {
		t.Fatalf("plan Debian target: %v", err)
	}
	if !strings.HasPrefix(debianPlan.Kernel.URL, "https://mirror.example/debian/") {
		t.Fatalf("Debian kernel URL = %q", debianPlan.Kernel.URL)
	}

	ubuntuPlan, err := registry.Plan(context.Background(), provider.Target{
		ID:         "ubuntu-2604-amd64-netboot",
		ProviderID: "ubuntu",
	})
	if err != nil {
		t.Fatalf("plan Ubuntu target: %v", err)
	}
	if !strings.HasPrefix(ubuntuPlan.Kernel.URL, "https://releases.example/26.04/") {
		t.Fatalf("Ubuntu kernel URL = %q", ubuntuPlan.Kernel.URL)
	}
	if ubuntuPlan.Kernel.SHA256 != strings.Repeat("a", 64) {
		t.Fatalf("Ubuntu kernel sha256 = %q", ubuntuPlan.Kernel.SHA256)
	}
	if ubuntuPlan.Initrd.SHA256 != strings.Repeat("b", 64) {
		t.Fatalf("Ubuntu initrd sha256 = %q", ubuntuPlan.Initrd.SHA256)
	}
}
