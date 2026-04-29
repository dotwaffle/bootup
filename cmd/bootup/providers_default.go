//go:build !bootup_debian_fixture

package main

import (
	"fmt"

	"github.com/dotwaffle/bootup/internal/provider"
	"github.com/dotwaffle/bootup/internal/providers/debian"
	"github.com/dotwaffle/bootup/internal/trustmaterial"
)

func registerProviders(registry *provider.Registry) error {
	if err := registry.Register(debian.NewProvider(debian.Config{
		Keyring: trustmaterial.DebianArchiveKeyring(),
	})); err != nil {
		return fmt.Errorf("register Debian provider: %w", err)
	}
	return nil
}
