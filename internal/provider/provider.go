// Package provider defines build-time boot target providers.
package provider

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ErrDuplicateProvider is returned when two providers use the same ID.
var ErrDuplicateProvider = errors.New("duplicate provider")

// ErrProviderNotFound is returned when a target references an unknown provider.
var ErrProviderNotFound = errors.New("provider not found")

// ErrStagingNotSupported is returned when a provider cannot stage artifacts.
var ErrStagingNotSupported = errors.New("staging not supported")

// ErrDiscoveryFamilyNotFound is returned when discovery is requested for an
// unknown provider family.
var ErrDiscoveryFamilyNotFound = errors.New("discovery family not found")

// ErrInvalidTarget is returned when a provider exposes malformed target
// metadata.
var ErrInvalidTarget = errors.New("invalid target")

// ErrInvalidDiscoveryFamily is returned when a provider exposes malformed
// discovery family metadata.
var ErrInvalidDiscoveryFamily = errors.New("invalid discovery family")

// CatalogEntry describes static catalog metadata for a concrete boot target.
type CatalogEntry struct {
	Distribution string `json:"distribution"`
	Release      string `json:"release"`
	Architecture string `json:"architecture"`
	Kind         string `json:"kind"`
}

// SourceEntry describes provider source metadata for a concrete boot target.
type SourceEntry struct {
	BaseURL string `json:"base_url,omitempty"`
	ISOName string `json:"iso_name,omitempty"`
}

// LifecycleStatus describes informational lifecycle decoration for a target.
type LifecycleStatus string

const (
	// LifecycleSupported means the provider believes the target is currently
	// supported.
	LifecycleSupported LifecycleStatus = "supported"

	// LifecycleObsolete means the provider believes the target is superseded
	// but not necessarily unavailable.
	LifecycleObsolete LifecycleStatus = "obsolete"

	// LifecycleEOL means the provider believes the target is end-of-life.
	LifecycleEOL LifecycleStatus = "eol"

	// LifecycleUnknown means the provider could not determine lifecycle state.
	LifecycleUnknown LifecycleStatus = "unknown"
)

// LifecycleEntry describes optional informational lifecycle decoration.
type LifecycleEntry struct {
	Status LifecycleStatus `json:"status,omitempty"`
	Source string          `json:"source,omitempty"`
	Date   string          `json:"date,omitempty"`
}

// Target describes an operating system installer or live environment that
// bootup can prepare and hand off to.
type Target struct {
	ID         string         `json:"id"`
	ProviderID string         `json:"provider_id"`
	Name       string         `json:"name"`
	Catalog    CatalogEntry   `json:"catalog"`
	Source     SourceEntry    `json:"source,omitzero"`
	Lifecycle  LifecycleEntry `json:"lifecycle,omitzero"`
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

// DiscoveryFamily describes a compiled-in provider family that can discover
// concrete targets at runtime.
type DiscoveryFamily struct {
	ID          string `json:"id"`
	ProviderID  string `json:"provider_id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// Discoverer is implemented by providers that can discover targets at runtime.
type Discoverer interface {
	DiscoveryFamily() DiscoveryFamily
	DiscoverTargets(context.Context) ([]Target, error)
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
			if err := ValidateTarget(id, target); err != nil {
				return nil, fmt.Errorf("list targets for %s: %w", id, err)
			}
		}
		targets = append(targets, providerTargets...)
	}
	return targets, nil
}

// DiscoveryFamilies returns discovery-capable provider families without
// running discovery.
func (r *Registry) DiscoveryFamilies() ([]DiscoveryFamily, error) {
	ids := make([]string, 0, len(r.providers))
	for id := range r.providers {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	var families []DiscoveryFamily
	for _, id := range ids {
		discoverer, ok := r.providers[id].(Discoverer)
		if !ok {
			continue
		}
		family := discoverer.DiscoveryFamily()
		if err := ValidateDiscoveryFamily(id, family); err != nil {
			return nil, fmt.Errorf("list discovery family for %s: %w", id, err)
		}
		families = append(families, family)
	}
	return families, nil
}

// DiscoverTargets returns discovered concrete targets for familyID.
func (r *Registry) DiscoverTargets(ctx context.Context, familyID string) ([]Target, error) {
	ids := make([]string, 0, len(r.providers))
	for id := range r.providers {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		discoverer, ok := r.providers[id].(Discoverer)
		if !ok {
			continue
		}
		family := discoverer.DiscoveryFamily()
		if err := ValidateDiscoveryFamily(id, family); err != nil {
			return nil, fmt.Errorf("discover targets for %s: %w", id, err)
		}
		if family.ID != familyID {
			continue
		}
		targets, err := discoverer.DiscoverTargets(ctx)
		if err != nil {
			return nil, fmt.Errorf("discover targets for %s: %w", familyID, err)
		}
		for _, target := range targets {
			if err := ValidateTarget(id, target); err != nil {
				return nil, fmt.Errorf("discover targets for %s: %w", familyID, err)
			}
		}
		return targets, nil
	}
	return nil, fmt.Errorf("%w: %s", ErrDiscoveryFamilyNotFound, familyID)
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

// ValidateDiscoveryFamily validates discovery family metadata exposed by a
// provider.
func ValidateDiscoveryFamily(providerID string, family DiscoveryFamily) error {
	if strings.TrimSpace(family.ID) == "" {
		return fmt.Errorf("%w: provider %s returned family with empty ID", ErrInvalidDiscoveryFamily, providerID)
	}
	if strings.TrimSpace(family.ProviderID) == "" {
		return fmt.Errorf("%w: family %s has empty provider ID", ErrInvalidDiscoveryFamily, family.ID)
	}
	if family.ProviderID != providerID {
		return fmt.Errorf("%w: family %s provider ID %q does not match %q", ErrInvalidDiscoveryFamily, family.ID, family.ProviderID, providerID)
	}
	if strings.TrimSpace(family.Name) == "" {
		return fmt.Errorf("%w: family %s has empty name", ErrInvalidDiscoveryFamily, family.ID)
	}
	return nil
}

// ValidateTarget validates the metadata for a target exposed by providerID.
func ValidateTarget(providerID string, target Target) error {
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
	if err := validateSourceEntry(target.ID, target.Source); err != nil {
		return err
	}
	if err := validateLifecycleEntry(target.ID, target.Lifecycle); err != nil {
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

func validateSourceEntry(targetID string, source SourceEntry) error {
	if strings.TrimSpace(source.BaseURL) != source.BaseURL {
		return fmt.Errorf("%w: target %s source base URL has surrounding whitespace", ErrInvalidTarget, targetID)
	}
	if source.BaseURL != "" {
		parsed, err := url.Parse(source.BaseURL)
		if err != nil {
			return fmt.Errorf("%w: target %s source base URL is invalid: %w", ErrInvalidTarget, targetID, err)
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return fmt.Errorf("%w: target %s source base URL must use http or https", ErrInvalidTarget, targetID)
		}
		if parsed.Host == "" {
			return fmt.Errorf("%w: target %s source base URL must include host", ErrInvalidTarget, targetID)
		}
	}
	if strings.TrimSpace(source.ISOName) != source.ISOName {
		return fmt.Errorf("%w: target %s source ISO name has surrounding whitespace", ErrInvalidTarget, targetID)
	}
	if source.ISOName != "" {
		if strings.ContainsAny(source.ISOName, `/\`) || filepath.Base(source.ISOName) != source.ISOName {
			return fmt.Errorf("%w: target %s source ISO name must be a filename", ErrInvalidTarget, targetID)
		}
	}
	return nil
}

func validateLifecycleEntry(targetID string, lifecycle LifecycleEntry) error {
	if lifecycle == (LifecycleEntry{}) {
		return nil
	}
	switch lifecycle.Status {
	case LifecycleSupported, LifecycleObsolete, LifecycleEOL, LifecycleUnknown:
	case "":
		return fmt.Errorf("%w: target %s lifecycle status is empty", ErrInvalidTarget, targetID)
	default:
		return fmt.Errorf("%w: target %s lifecycle status %q is invalid", ErrInvalidTarget, targetID, lifecycle.Status)
	}
	if strings.TrimSpace(lifecycle.Source) != lifecycle.Source {
		return fmt.Errorf("%w: target %s lifecycle source has surrounding whitespace", ErrInvalidTarget, targetID)
	}
	if strings.TrimSpace(lifecycle.Date) != lifecycle.Date {
		return fmt.Errorf("%w: target %s lifecycle date has surrounding whitespace", ErrInvalidTarget, targetID)
	}
	if lifecycle.Date != "" {
		if _, err := time.Parse(time.DateOnly, lifecycle.Date); err != nil {
			return fmt.Errorf("%w: target %s lifecycle date must use YYYY-MM-DD", ErrInvalidTarget, targetID)
		}
	}
	return nil
}
