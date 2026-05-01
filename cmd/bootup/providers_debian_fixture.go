//go:build bootup_debian_fixture

package main

import (
	"fmt"

	"github.com/dotwaffle/bootup/internal/catalog"
	"github.com/dotwaffle/bootup/internal/provider"
	"github.com/dotwaffle/bootup/internal/providerconfig"
	"github.com/dotwaffle/bootup/internal/providers/debianfixture"
	"github.com/dotwaffle/bootup/internal/providers/fedora"
	"github.com/dotwaffle/bootup/internal/providers/linux"
	"github.com/dotwaffle/bootup/internal/providers/localdisk"
	"github.com/dotwaffle/bootup/internal/providers/ubuntu"
)

func compiledProviderIDs() []string {
	return []string{"debian", "fedora", "linux", "local", "ubuntu"}
}

func registerProviders(registry *provider.Registry, _ providerconfig.Config, catalogDoc catalog.Document) error {
	p, err := debianfixture.NewProvider(catalogDoc.Targets("debian"))
	if err != nil {
		return fmt.Errorf("create Debian fixture provider: %w", err)
	}
	if err := registry.Register(p); err != nil {
		return fmt.Errorf("register Debian fixture provider: %w", err)
	}
	if err := registry.Register(ubuntu.NewProvider(ubuntu.Config{
		Targets: catalogDoc.Targets("ubuntu"),
	})); err != nil {
		return fmt.Errorf("register Ubuntu provider: %w", err)
	}
	if err := registry.Register(fedora.NewProvider(fedora.Config{
		Targets: catalogDoc.Targets("fedora"),
	})); err != nil {
		return fmt.Errorf("register Fedora provider: %w", err)
	}
	if err := registry.Register(linux.NewProvider(linux.Config{
		Targets: catalogDoc.Targets("linux"),
	})); err != nil {
		return fmt.Errorf("register Linux provider: %w", err)
	}
	if err := registry.Register(localdisk.NewProvider(localdisk.Config{
		Targets: catalogDoc.Targets("local"),
	})); err != nil {
		return fmt.Errorf("register local disk provider: %w", err)
	}
	return nil
}
