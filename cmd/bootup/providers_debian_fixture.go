//go:build bootup_debian_fixture

package main

import (
	"fmt"

	"github.com/dotwaffle/bootup/internal/provider"
	"github.com/dotwaffle/bootup/internal/providerconfig"
	"github.com/dotwaffle/bootup/internal/providers/debianfixture"
)

func registerProviders(registry *provider.Registry, _ providerconfig.Config) error {
	p, err := debianfixture.NewProvider()
	if err != nil {
		return fmt.Errorf("create Debian fixture provider: %w", err)
	}
	if err := registry.Register(p); err != nil {
		return fmt.Errorf("register Debian fixture provider: %w", err)
	}
	return nil
}
