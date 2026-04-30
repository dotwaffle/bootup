//go:build !bootup_debian_fixture

package main

import (
	"fmt"

	"github.com/dotwaffle/bootup/internal/provider"
	"github.com/dotwaffle/bootup/internal/providerconfig"
	"github.com/dotwaffle/bootup/internal/providers/debian"
	"github.com/dotwaffle/bootup/internal/providers/ubuntu"
)

func registerProviders(registry *provider.Registry, config providerconfig.Config) error {
	if err := registry.Register(debian.NewProvider(debian.Config{
		MirrorURL: config.Debian.MirrorURL,
		Keyring:   config.Debian.Keyring,
	})); err != nil {
		return fmt.Errorf("register Debian provider: %w", err)
	}
	if err := registry.Register(ubuntu.NewProvider(ubuntu.Config{
		ReleaseURL:   config.Ubuntu.ReleaseURL,
		Keyring:      config.Ubuntu.Keyring,
		KernelSHA256: config.Ubuntu.KernelSHA256,
		InitrdSHA256: config.Ubuntu.InitrdSHA256,
	})); err != nil {
		return fmt.Errorf("register Ubuntu provider: %w", err)
	}
	return nil
}
