//go:build !bootup_debian_fixture

package main

import (
	"context"
	"testing"

	"github.com/dotwaffle/bootup/internal/provider"
)

func TestRegisterProvidersIncludesUbuntu(t *testing.T) {
	t.Parallel()

	registry := provider.NewRegistry()
	if err := registerProviders(registry); err != nil {
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
