// Package provider defines build-time boot target providers.
package provider

import (
	"context"
	"errors"
	"fmt"
	"sort"
)

// ErrDuplicateProvider is returned when two providers use the same ID.
var ErrDuplicateProvider = errors.New("duplicate provider")

// ErrProviderNotFound is returned when a target references an unknown provider.
var ErrProviderNotFound = errors.New("provider not found")

// Target describes an operating system installer or live environment that
// bootup can prepare and hand off to.
type Target struct {
	ID           string
	ProviderID   string
	Name         string
	Architecture string
}

// Artifact describes a boot artifact that can be downloaded and verified.
type Artifact struct {
	Name   string
	URL    string
	SHA256 string
	Path   string
}

// Verification describes metadata required to trust boot artifacts.
type Verification struct {
	MetadataURL  string
	ChecksumURL  string
	SignatureURL string
}

// BootPlan describes the artifacts and command line required for kexec.
type BootPlan struct {
	Target       Target
	Kernel       Artifact
	Initrd       Artifact
	Cmdline      string
	Verification Verification
}

// Provider exposes boot targets and plans for a distribution or tool family.
type Provider interface {
	ID() string
	Targets(context.Context) ([]Target, error)
	Plan(context.Context, Target) (BootPlan, error)
}

// Registry stores build-time providers compiled into the bootup image.
type Registry struct {
	providers map[string]Provider
}

// NewRegistry creates an empty provider registry.
func NewRegistry() *Registry {
	return &Registry{providers: make(map[string]Provider)}
}

// Register adds a provider to the registry.
func (r *Registry) Register(provider Provider) error {
	id := provider.ID()
	if _, exists := r.providers[id]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicateProvider, id)
	}
	r.providers[id] = provider
	return nil
}

// Targets returns every target exposed by registered providers.
func (r *Registry) Targets(ctx context.Context) ([]Target, error) {
	ids := make([]string, 0, len(r.providers))
	for id := range r.providers {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	var targets []Target
	for _, id := range ids {
		providerTargets, err := r.providers[id].Targets(ctx)
		if err != nil {
			return nil, fmt.Errorf("list targets for %s: %w", id, err)
		}
		targets = append(targets, providerTargets...)
	}
	return targets, nil
}

// Plan returns the boot plan for target from its provider.
func (r *Registry) Plan(ctx context.Context, target Target) (BootPlan, error) {
	provider, ok := r.providers[target.ProviderID]
	if !ok {
		return BootPlan{}, fmt.Errorf("%w: %s", ErrProviderNotFound, target.ProviderID)
	}
	plan, err := provider.Plan(ctx, target)
	if err != nil {
		return BootPlan{}, fmt.Errorf("plan target %s: %w", target.ID, err)
	}
	return plan, nil
}
