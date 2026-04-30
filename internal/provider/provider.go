// Package provider defines build-time boot target providers.
package provider

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// ErrDuplicateProvider is returned when two providers use the same ID.
var ErrDuplicateProvider = errors.New("duplicate provider")

// ErrProviderNotFound is returned when a target references an unknown provider.
var ErrProviderNotFound = errors.New("provider not found")

// ErrStagingNotSupported is returned when a provider cannot stage artifacts.
var ErrStagingNotSupported = errors.New("staging not supported")

// ErrInvalidTarget is returned when a provider exposes malformed target
// metadata.
var ErrInvalidTarget = errors.New("invalid target")

// CatalogEntry describes static catalog metadata for a concrete boot target.
type CatalogEntry struct {
	Distribution string
	Release      string
	Architecture string
	Kind         string
}

// Target describes an operating system installer or live environment that
// bootup can prepare and hand off to.
type Target struct {
	ID         string
	ProviderID string
	Name       string
	Catalog    CatalogEntry
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

// StageConfig configures provider-specific artifact staging.
type StageConfig struct {
	Plan       BootPlan
	StagingDir string
}

// Provider exposes boot targets and plans for a distribution or tool family.
type Provider interface {
	ID() string
	Targets(context.Context) ([]Target, error)
	Plan(context.Context, Target) (BootPlan, error)
}

// Stager stages and verifies artifacts for a planned boot target.
type Stager interface {
	Stage(context.Context, StageConfig) (BootPlan, error)
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
		for _, target := range providerTargets {
			if err := validateProviderTarget(id, target); err != nil {
				return nil, fmt.Errorf("list targets for %s: %w", id, err)
			}
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

// Stage stages and verifies artifacts for plan through its provider.
func (r *Registry) Stage(ctx context.Context, config StageConfig) (BootPlan, error) {
	provider, ok := r.providers[config.Plan.Target.ProviderID]
	if !ok {
		return BootPlan{}, fmt.Errorf("%w: %s", ErrProviderNotFound, config.Plan.Target.ProviderID)
	}
	stager, ok := provider.(Stager)
	if !ok {
		return BootPlan{}, fmt.Errorf("%w: %s", ErrStagingNotSupported, provider.ID())
	}
	staged, err := stager.Stage(ctx, config)
	if err != nil {
		return BootPlan{}, fmt.Errorf("stage target %s: %w", config.Plan.Target.ID, err)
	}
	return staged, nil
}

func validateProviderTarget(providerID string, target Target) error {
	if strings.TrimSpace(target.ID) == "" {
		return fmt.Errorf("%w: provider %s returned target with empty ID", ErrInvalidTarget, providerID)
	}
	if strings.TrimSpace(target.ProviderID) == "" {
		return fmt.Errorf("%w: target %s has empty provider ID", ErrInvalidTarget, target.ID)
	}
	if target.ProviderID != providerID {
		return fmt.Errorf("%w: target %s provider ID %q does not match %q", ErrInvalidTarget, target.ID, target.ProviderID, providerID)
	}
	if strings.TrimSpace(target.Name) == "" {
		return fmt.Errorf("%w: target %s has empty name", ErrInvalidTarget, target.ID)
	}
	if err := validateCatalogEntry(target.ID, target.Catalog); err != nil {
		return err
	}
	return nil
}

func validateCatalogEntry(targetID string, catalog CatalogEntry) error {
	if strings.TrimSpace(catalog.Distribution) == "" {
		return fmt.Errorf("%w: target %s catalog distribution is empty", ErrInvalidTarget, targetID)
	}
	if strings.TrimSpace(catalog.Release) == "" {
		return fmt.Errorf("%w: target %s catalog release is empty", ErrInvalidTarget, targetID)
	}
	if strings.TrimSpace(catalog.Architecture) == "" {
		return fmt.Errorf("%w: target %s catalog architecture is empty", ErrInvalidTarget, targetID)
	}
	if strings.TrimSpace(catalog.Kind) == "" {
		return fmt.Errorf("%w: target %s catalog kind is empty", ErrInvalidTarget, targetID)
	}
	return nil
}
