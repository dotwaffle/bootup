//go:build !bootup_debian_fixture

package main

import (
	"fmt"

	"github.com/dotwaffle/bootup/internal/catalog"
	"github.com/dotwaffle/bootup/internal/provider"
	"github.com/dotwaffle/bootup/internal/providerconfig"
	"github.com/dotwaffle/bootup/internal/providers/debian"
	"github.com/dotwaffle/bootup/internal/providers/ubuntu"
)

func compiledProviderIDs() []string {
	return []string{"debian", "ubuntu"}
}

func registerProviders(registry *provider.Registry, config providerconfig.Config, catalogDoc catalog.Document) error {
	if err := registry.Register(debian.NewProvider(debian.Config{
		MirrorURL: config.Debian.MirrorURL,
		Keyring:   config.Debian.Keyring,
		Targets:   catalogDoc.Targets("debian"),
	})); err != nil {
		return fmt.Errorf("register Debian provider: %w", err)
	}
	if err := registry.Register(ubuntu.NewProvider(ubuntu.Config{
		ReleaseURL:   config.Ubuntu.ReleaseURL,
		Keyring:      config.Ubuntu.Keyring,
		KernelSHA256: config.Ubuntu.KernelSHA256,
		InitrdSHA256: config.Ubuntu.InitrdSHA256,
		Targets:      catalogDoc.Targets("ubuntu"),
	})); err != nil {
		return fmt.Errorf("register Ubuntu provider: %w", err)
	}
	return nil
}
