// Package localdisk provides a local disk boot target.
package localdisk

import (
	"context"
	"fmt"

	"github.com/dotwaffle/bootup/internal/provider"
)

const (
	providerID = "local"
	targetID   = "local-disk-auto"
)

// Config configures the local disk provider.
type Config struct {
	Targets []provider.Target
}

// Provider exposes local disk boot targets.
type Provider struct {
	targets []provider.Target
}

// NewProvider creates a local disk provider.
func NewProvider(config Config) *Provider {
	targets := cloneTargets(config.Targets)
	if config.Targets == nil {
		targets = defaultTargets()
	}
	return &Provider{targets: targets}
}

// ID returns the provider ID.
func (*Provider) ID() string {
	return providerID
}

// Targets returns local disk boot targets.
func (p *Provider) Targets(context.Context) ([]provider.Target, error) {
	return cloneTargets(p.targets), nil
}

// Plan returns a local boot plan for target.
func (p *Provider) Plan(_ context.Context, input provider.PlanInput) (provider.BootPlan, error) {
	target := input.Target
	selected, err := p.selectedTarget(target)
	if err != nil {
		return provider.BootPlan{}, err
	}
	if provider.ResolveBootAction(selected.Action) != provider.BootActionLocalBoot {
		return provider.BootPlan{}, fmt.Errorf("unsupported local boot action %q for target %s", selected.Action, selected.ID)
	}
	plan := provider.BootPlan{
		Target: selected,
		Action: provider.BootActionLocalBoot,
	}
	return provider.ApplySelectedOptions(plan, input.Options)
}

// Stage returns the local boot plan unchanged.
func (*Provider) Stage(_ context.Context, config provider.StageConfig) (provider.BootPlan, error) {
	return config.Plan, nil
}

func (p *Provider) selectedTarget(target provider.Target) (provider.Target, error) {
	if selected, ok := p.target(target.ID); ok {
		return selected, nil
	}
	if err := provider.ValidateTarget(providerID, target); err != nil {
		return provider.Target{}, err
	}
	return target, nil
}

func (p *Provider) target(id string) (provider.Target, bool) {
	for _, target := range p.targets {
		if target.ID == id {
			return target, true
		}
	}
	return provider.Target{}, false
}

func defaultTargets() []provider.Target {
	return []provider.Target{{
		ID:         targetID,
		ProviderID: providerID,
		Name:       "Boot from local disk",
		Action:     provider.BootActionLocalBoot,
		Catalog: provider.CatalogEntry{
			Distribution: "local",
			Release:      "disk",
			Architecture: "amd64",
			Kind:         "localboot",
		},
	}}
}

func cloneTargets(targets []provider.Target) []provider.Target {
	return append([]provider.Target(nil), targets...)
}
