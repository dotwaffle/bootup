//go:build !bootup_debian_fixture

package main

import (
	"context"
	"strings"
	"testing"

	"github.com/dotwaffle/bootup/internal/provider"
	"github.com/dotwaffle/bootup/internal/providerconfig"
)

func TestRegisterProvidersIncludesUbuntu(t *testing.T) {
	t.Parallel()

	registry := provider.NewRegistry()
	if err := registerProviders(registry, providerconfig.Config{}); err != nil {
		t.Fatalf("register providers: %v", err)
	}
	targets, err := registry.Targets(context.Background())
	if err != nil {
		t.Fatalf("targets: %v", err)
	}
	for _, target := range targets {
		if target.ID == "ubuntu-2604-amd64-netboot" {
			return
		}
	}
	t.Fatalf("registered targets = %#v, want Ubuntu target", targets)
}

func TestRegisterProvidersAppliesRuntimeConfig(t *testing.T) {
	t.Parallel()

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
	}); err != nil {
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
