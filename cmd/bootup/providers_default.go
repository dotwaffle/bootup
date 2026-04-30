//go:build !bootup_debian_fixture

package main

import (
	"fmt"

	"github.com/dotwaffle/bootup/internal/catalog"
	"github.com/dotwaffle/bootup/internal/provider"
	"github.com/dotwaffle/bootup/internal/providerconfig"
	"github.com/dotwaffle/bootup/internal/providers/debian"
	"github.com/dotwaffle/bootup/internal/providers/fedora"
	"github.com/dotwaffle/bootup/internal/providers/ubuntu"
)

func compiledProviderIDs() []string {
	return []string{"debian", "fedora", "ubuntu"}
}

func registerProviders(registry *provider.Registry, config providerconfig.Config, catalogDoc catalog.Document) error {
	if err := registry.Register(debian.NewProvider(debian.Config{
		MirrorURL:        config.Debian.MirrorURL,
		DiscoveryURL:     config.Debian.DiscoveryURL,
		Keyring:          config.Debian.Keyring,
		Targets:          catalogDoc.Targets("debian"),
		DiscoveryTimeout: config.Debian.DiscoveryTimeout,
		Lifecycle:        config.Debian.Lifecycle,
	})); err != nil {
		return fmt.Errorf("register Debian provider: %w", err)
	}
	if err := registry.Register(ubuntu.NewProvider(ubuntu.Config{
		ReleaseURL:       config.Ubuntu.ReleaseURL,
		DiscoveryURL:     config.Ubuntu.DiscoveryURL,
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
		ReleaseURL:   config.Fedora.ReleaseURL,
		KernelSHA256: config.Fedora.KernelSHA256,
		InitrdSHA256: config.Fedora.InitrdSHA256,
		Targets:      catalogDoc.Targets("fedora"),
	})); err != nil {
		return fmt.Errorf("register Fedora provider: %w", err)
	}
	return nil
}
