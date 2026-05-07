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
	"github.com/dotwaffle/bootup/internal/providers/mfsbsd"
	"github.com/dotwaffle/bootup/internal/providers/ubuntu"
)

func compiledProviderIDs() []string {
	return []string{"debian", "fedora", "linux", "local", "mfsbsd", "ubuntu"}
}

func registerProviders(registry *provider.Registry, config providerconfig.Config, catalogDoc catalog.Document) error {
	p, err := debianfixture.NewProvider(catalogDoc.Targets("debian"))
	if err != nil {
		return fmt.Errorf("create Debian fixture provider: %w", err)
	}
	if err := registry.Register(p); err != nil {
		return fmt.Errorf("register Debian fixture provider: %w", err)
	}
	if err := registry.Register(ubuntu.NewProvider(ubuntu.Config{
		ReleaseURL:       config.Ubuntu.ReleaseURL,
		DiscoveryURL:     config.Ubuntu.DiscoveryURL,
		DiscoveryFile:    config.Ubuntu.DiscoveryFile,
		Keyring:          config.Ubuntu.Keyring,
		KernelSHA256:     config.Ubuntu.KernelSHA256,
		InitrdSHA256:     config.Ubuntu.InitrdSHA256,
		Targets:          catalogDoc.Targets("ubuntu"),
		DiscoveryTimeout: config.Ubuntu.DiscoveryTimeout,
		Lifecycle:        config.Ubuntu.Lifecycle,
	})); err != nil {
		return fmt.Errorf("register Ubuntu provider: %w", err)
	}
	if err := registry.Register(fedora.NewProvider(fedora.Config{
		ReleaseURL:       config.Fedora.ReleaseURL,
		DiscoveryURL:     config.Fedora.DiscoveryURL,
		DiscoveryFile:    config.Fedora.DiscoveryFile,
		KernelSHA256:     config.Fedora.KernelSHA256,
		InitrdSHA256:     config.Fedora.InitrdSHA256,
		Targets:          catalogDoc.Targets("fedora"),
		DiscoveryTimeout: config.Fedora.DiscoveryTimeout,
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
	if err := registry.Register(mfsbsd.NewProvider(mfsbsd.Config{
		Targets: catalogDoc.Targets("mfsbsd"),
	})); err != nil {
		return fmt.Errorf("register mfsBSD provider: %w", err)
	}
	return nil
}
