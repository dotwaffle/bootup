// Package provider defines build-time boot target providers.
package provider

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
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

// ErrInvalidTargetOption is returned when selected target options are invalid.
var ErrInvalidTargetOption = errors.New("invalid target option")

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
	BaseURL      string `json:"base_url,omitzero"`
	ISOName      string `json:"iso_name,omitzero"`
	ISOSHA256    string `json:"iso_sha256,omitzero"`
	KernelPath   string `json:"kernel_path,omitzero"`
	InitrdPath   string `json:"initrd_path,omitzero"`
	KernelSHA256 string `json:"kernel_sha256,omitzero"`
	InitrdSHA256 string `json:"initrd_sha256,omitzero"`
	Cmdline      string `json:"cmdline,omitzero"`
}

// BootAction describes how bootup hands off to a selected target.
type BootAction string

const (
	// BootActionLinuxKexec stages Linux boot artifacts and executes kexec.
	BootActionLinuxKexec BootAction = "linux-kexec"

	// BootActionLocalBoot runs the local disk boot path.
	BootActionLocalBoot BootAction = "localboot"

	// BootActionFreeBSDKboot runs FreeBSD loader.kboot from Linux stage-1.
	BootActionFreeBSDKboot BootAction = "freebsd-kboot"
)

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
	Status LifecycleStatus `json:"status,omitzero"`
	Source string          `json:"source,omitzero"`
	Date   string          `json:"date,omitzero"`
}

// TargetOptionType identifies how a target option value is validated.
type TargetOptionType string

const (
	// TargetOptionBool appends a fixed fragment when selected true.
	TargetOptionBool TargetOptionType = "bool"

	// TargetOptionEnum selects one of a fixed set of allowed values.
	TargetOptionEnum TargetOptionType = "enum"

	// TargetOptionString expands a selected string value through a template.
	TargetOptionString TargetOptionType = "string"
)

// TargetOptionValue describes one allowed enum value.
type TargetOptionValue struct {
	Value    string `json:"value"`
	Label    string `json:"label,omitzero"`
	Fragment string `json:"fragment,omitzero"`
}

// TargetOption describes one operator-selectable target option.
type TargetOption struct {
	ID       string              `json:"id"`
	Label    string              `json:"label"`
	Type     TargetOptionType    `json:"type"`
	Fragment string              `json:"fragment,omitzero"`
	Values   []TargetOptionValue `json:"values,omitzero"`
	Template string              `json:"template,omitzero"`
}

// SelectedOption describes an operator-selected target option value.
type SelectedOption struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

// Target describes an operating system installer or live environment that
// bootup can prepare and hand off to.
type Target struct {
	ID         string         `json:"id"`
	ProviderID string         `json:"provider_id"`
	Name       string         `json:"name"`
	Action     BootAction     `json:"action,omitzero"`
	Catalog    CatalogEntry   `json:"catalog"`
	Source     SourceEntry    `json:"source,omitzero"`
	Lifecycle  LifecycleEntry `json:"lifecycle,omitzero"`
	Options    []TargetOption `json:"options,omitzero"`
}

// PlanInput describes an explicit provider planning request.
type PlanInput struct {
	Target  Target
	Options []SelectedOption
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

// FreeBSDKbootPlan describes a staged FreeBSD loader.kboot handoff.
type FreeBSDKbootPlan struct {
	Loader        Artifact
	LoaderHelp    Artifact
	LoaderArchive Artifact
	Payload       Artifact
	PayloadRoot   string
	Args          []string
}

// BootPlan describes the artifacts and command line required for kexec.
type BootPlan struct {
	Target       Target
	Action       BootAction
	Kernel       Artifact
	Initrd       Artifact
	Cmdline      string
	Verification Verification
	FreeBSDKboot FreeBSDKbootPlan
}

// ResolvedAction returns the plan action, defaulting old plans to Linux kexec.
func (p BootPlan) ResolvedAction() BootAction {
	return ResolveBootAction(p.Action)
}

// ResolveBootAction returns action, defaulting an empty value to Linux kexec.
func ResolveBootAction(action BootAction) BootAction {
	if action == "" {
		return BootActionLinuxKexec
	}
	return action
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
	Plan(context.Context, PlanInput) (BootPlan, error)
}

// DiscoveryFamily describes a compiled-in provider family that can discover
// concrete targets at runtime.
type DiscoveryFamily struct {
	ID          string `json:"id"`
	ProviderID  string `json:"provider_id"`
	Name        string `json:"name"`
	Description string `json:"description,omitzero"`
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
func (r *Registry) Plan(ctx context.Context, input PlanInput) (BootPlan, error) {
	if err := ValidateSelectedOptions(input.Target, input.Options); err != nil {
		return BootPlan{}, err
	}
	provider, ok := r.providers[input.Target.ProviderID]
	if !ok {
		return BootPlan{}, fmt.Errorf("%w: %s", ErrProviderNotFound, input.Target.ProviderID)
	}
	plan, err := provider.Plan(ctx, input)
	if err != nil {
		return BootPlan{}, fmt.Errorf("plan target %s: %w", input.Target.ID, err)
	}
	return plan, nil
}

// ValidateSelectedOptions validates selected option values against target.
func ValidateSelectedOptions(target Target, selected []SelectedOption) error {
	_, err := selectedOptionFragments(target, selected)
	return err
}

// ApplySelectedOptions appends selected option command-line fragments to plan.
func ApplySelectedOptions(plan BootPlan, selected []SelectedOption) (BootPlan, error) {
	fragments, err := selectedOptionFragments(plan.Target, selected)
	if err != nil {
		return BootPlan{}, err
	}
	switch plan.ResolvedAction() {
	case BootActionFreeBSDKboot:
		plan.FreeBSDKboot.Args = append(plan.FreeBSDKboot.Args, fragments...)
	default:
		plan.Cmdline = appendCmdline(plan.Cmdline, strings.Join(fragments, " "))
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
	if err := validateBootAction(target.ID, target.Action); err != nil {
		return err
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
	if err := validateTargetOptions(target.ID, target.Options); err != nil {
		return err
	}
	return nil
}

func validateBootAction(targetID string, action BootAction) error {
	switch ResolveBootAction(action) {
	case BootActionLinuxKexec, BootActionLocalBoot, BootActionFreeBSDKboot:
		return nil
	default:
		return fmt.Errorf("%w: target %s boot action %q is invalid", ErrInvalidTarget, targetID, action)
	}
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
	if strings.TrimSpace(source.ISOSHA256) != source.ISOSHA256 {
		return fmt.Errorf("%w: target %s source ISO SHA256 has surrounding whitespace", ErrInvalidTarget, targetID)
	}
	if err := validateSourcePath(targetID, "kernel path", source.KernelPath); err != nil {
		return err
	}
	if err := validateSourcePath(targetID, "initrd path", source.InitrdPath); err != nil {
		return err
	}
	if err := validateSourceHashPins(targetID, source); err != nil {
		return err
	}
	if strings.TrimSpace(source.Cmdline) != source.Cmdline {
		return fmt.Errorf("%w: target %s source cmdline has surrounding whitespace", ErrInvalidTarget, targetID)
	}
	return nil
}

func validateSourceHashPins(targetID string, source SourceEntry) error {
	if strings.TrimSpace(source.KernelSHA256) != source.KernelSHA256 {
		return fmt.Errorf("%w: target %s source kernel SHA256 has surrounding whitespace", ErrInvalidTarget, targetID)
	}
	if strings.TrimSpace(source.InitrdSHA256) != source.InitrdSHA256 {
		return fmt.Errorf("%w: target %s source initrd SHA256 has surrounding whitespace", ErrInvalidTarget, targetID)
	}
	if source.KernelSHA256 != "" {
		if source.KernelPath == "" {
			return fmt.Errorf("%w: target %s source kernel SHA256 requires kernel path", ErrInvalidTarget, targetID)
		}
		if err := validateSourceSHA256(targetID, "kernel", source.KernelSHA256); err != nil {
			return err
		}
	}
	if source.InitrdSHA256 != "" {
		if source.InitrdPath == "" {
			return fmt.Errorf("%w: target %s source initrd SHA256 requires initrd path", ErrInvalidTarget, targetID)
		}
		if err := validateSourceSHA256(targetID, "initrd", source.InitrdSHA256); err != nil {
			return err
		}
	}
	if source.InitrdPath != "" && (source.KernelSHA256 == "") != (source.InitrdSHA256 == "") {
		return fmt.Errorf("%w: target %s source kernel and initrd SHA256 pins must be supplied together", ErrInvalidTarget, targetID)
	}
	return nil
}

func validateSourceSHA256(targetID string, name string, value string) error {
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != 32 {
		return fmt.Errorf("%w: target %s source %s SHA256 must be a 64-character SHA-256 hex digest", ErrInvalidTarget, targetID, name)
	}
	return nil
}

func validateSourcePath(targetID string, name string, value string) error {
	if strings.TrimSpace(value) != value {
		return fmt.Errorf("%w: target %s source %s has surrounding whitespace", ErrInvalidTarget, targetID, name)
	}
	if value == "" {
		return nil
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("%w: target %s source %s is invalid: %w", ErrInvalidTarget, targetID, name, err)
	}
	if parsed.IsAbs() || parsed.Host != "" {
		return fmt.Errorf("%w: target %s source %s must be relative", ErrInvalidTarget, targetID, name)
	}
	if strings.HasPrefix(value, "/") || strings.Contains(value, `\`) {
		return fmt.Errorf("%w: target %s source %s must be a clean relative URL path", ErrInvalidTarget, targetID, name)
	}
	if clean := path.Clean(value); clean == "." || clean != value || clean == ".." || strings.HasPrefix(clean, "../") {
		return fmt.Errorf("%w: target %s source %s must be a clean relative URL path", ErrInvalidTarget, targetID, name)
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

func validateTargetOptions(targetID string, options []TargetOption) error {
	seen := make(map[string]struct{}, len(options))
	for _, option := range options {
		if strings.TrimSpace(option.ID) == "" {
			return fmt.Errorf("%w: target %s option ID is empty", ErrInvalidTarget, targetID)
		}
		if !validOptionID(option.ID) {
			return fmt.Errorf("%w: target %s option %q ID is invalid", ErrInvalidTarget, targetID, option.ID)
		}
		if _, ok := seen[option.ID]; ok {
			return fmt.Errorf("%w: target %s option ID %q is duplicate", ErrInvalidTarget, targetID, option.ID)
		}
		seen[option.ID] = struct{}{}
		if strings.TrimSpace(option.Label) == "" {
			return fmt.Errorf("%w: target %s option %s label is empty", ErrInvalidTarget, targetID, option.ID)
		}
		if err := validateTargetOptionBehavior(targetID, option); err != nil {
			return err
		}
	}
	return nil
}

func selectedOptionFragments(target Target, selected []SelectedOption) ([]string, error) {
	if len(selected) == 0 {
		return nil, nil
	}
	selectedByID := make(map[string]SelectedOption, len(selected))
	for _, option := range selected {
		if strings.TrimSpace(option.ID) == "" {
			return nil, fmt.Errorf("%w: target %s selected option ID is empty", ErrInvalidTargetOption, target.ID)
		}
		if _, ok := selectedByID[option.ID]; ok {
			return nil, fmt.Errorf("%w: target %s selected option %q is duplicate", ErrInvalidTargetOption, target.ID, option.ID)
		}
		selectedByID[option.ID] = option
	}

	var fragments []string
	for _, option := range target.Options {
		selectedOption, ok := selectedByID[option.ID]
		if !ok {
			continue
		}
		delete(selectedByID, option.ID)
		fragment, err := selectedOptionFragment(target.ID, option, selectedOption)
		if err != nil {
			return nil, err
		}
		if fragment != "" {
			fragments = append(fragments, fragment)
		}
	}
	for id := range selectedByID {
		return nil, fmt.Errorf("%w: target %s does not declare option %q", ErrInvalidTargetOption, target.ID, id)
	}
	return fragments, nil
}

func selectedOptionFragment(targetID string, option TargetOption, selected SelectedOption) (string, error) {
	switch option.Type {
	case TargetOptionBool:
		switch selected.Value {
		case "true":
			return option.Fragment, nil
		case "false":
			return "", nil
		default:
			return "", fmt.Errorf("%w: target %s option %s value %q is not boolean", ErrInvalidTargetOption, targetID, option.ID, selected.Value)
		}
	case TargetOptionEnum:
		for _, value := range option.Values {
			if selected.Value == value.Value {
				return value.Fragment, nil
			}
		}
		return "", fmt.Errorf("%w: target %s option %s value %q is not allowed", ErrInvalidTargetOption, targetID, option.ID, selected.Value)
	case TargetOptionString:
		if strings.TrimSpace(selected.Value) == "" {
			return "", fmt.Errorf("%w: target %s option %s value is empty", ErrInvalidTargetOption, targetID, option.ID)
		}
		if strings.TrimSpace(selected.Value) != selected.Value || containsSpaceOrControl(selected.Value) {
			return "", fmt.Errorf("%w: target %s option %s value %q is invalid", ErrInvalidTargetOption, targetID, option.ID, selected.Value)
		}
		fragment := strings.ReplaceAll(option.Template, "{value}", selected.Value)
		if err := validateExpandedCmdlineFragment(targetID, option.ID, fragment); err != nil {
			return "", err
		}
		return fragment, nil
	default:
		return "", fmt.Errorf("%w: target %s option %s type %q is invalid", ErrInvalidTargetOption, targetID, option.ID, option.Type)
	}
}

func validateExpandedCmdlineFragment(targetID string, optionID string, fragment string) error {
	if err := validateCmdlineFragment(targetID, optionID, fragment, true); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidTargetOption, err)
	}
	return nil
}

func validateTargetOptionBehavior(targetID string, option TargetOption) error {
	switch option.Type {
	case TargetOptionBool:
		if len(option.Values) != 0 || option.Template != "" {
			return fmt.Errorf("%w: target %s bool option %s has incompatible fields", ErrInvalidTarget, targetID, option.ID)
		}
		return validateCmdlineFragment(targetID, option.ID, option.Fragment, true)
	case TargetOptionEnum:
		if option.Fragment != "" || option.Template != "" {
			return fmt.Errorf("%w: target %s enum option %s has incompatible fields", ErrInvalidTarget, targetID, option.ID)
		}
		return validateOptionValues(targetID, option)
	case TargetOptionString:
		if option.Fragment != "" || len(option.Values) != 0 {
			return fmt.Errorf("%w: target %s string option %s has incompatible fields", ErrInvalidTarget, targetID, option.ID)
		}
		return validateCmdlineTemplate(targetID, option.ID, option.Template)
	case "":
		return fmt.Errorf("%w: target %s option %s type is empty", ErrInvalidTarget, targetID, option.ID)
	default:
		return fmt.Errorf("%w: target %s option %s type %q is invalid", ErrInvalidTarget, targetID, option.ID, option.Type)
	}
}

func validateOptionValues(targetID string, option TargetOption) error {
	if len(option.Values) == 0 {
		return fmt.Errorf("%w: target %s enum option %s has no values", ErrInvalidTarget, targetID, option.ID)
	}
	seen := make(map[string]struct{}, len(option.Values))
	for _, value := range option.Values {
		if strings.TrimSpace(value.Value) == "" {
			return fmt.Errorf("%w: target %s enum option %s value is empty", ErrInvalidTarget, targetID, option.ID)
		}
		if strings.TrimSpace(value.Value) != value.Value || containsSpaceOrControl(value.Value) {
			return fmt.Errorf("%w: target %s enum option %s value %q is invalid", ErrInvalidTarget, targetID, option.ID, value.Value)
		}
		if _, ok := seen[value.Value]; ok {
			return fmt.Errorf("%w: target %s enum option %s value %q is duplicate", ErrInvalidTarget, targetID, option.ID, value.Value)
		}
		seen[value.Value] = struct{}{}
		if strings.TrimSpace(value.Label) != value.Label {
			return fmt.Errorf("%w: target %s enum option %s value %q label has surrounding whitespace", ErrInvalidTarget, targetID, option.ID, value.Value)
		}
		if err := validateCmdlineFragment(targetID, option.ID, value.Fragment, false); err != nil {
			return err
		}
	}
	return nil
}

func validateCmdlineFragment(targetID string, optionID string, fragment string, required bool) error {
	if fragment == "" {
		if required {
			return fmt.Errorf("%w: target %s option %s command-line fragment is empty", ErrInvalidTarget, targetID, optionID)
		}
		return nil
	}
	if strings.TrimSpace(fragment) != fragment {
		return fmt.Errorf("%w: target %s option %s command-line fragment has surrounding whitespace", ErrInvalidTarget, targetID, optionID)
	}
	if strings.Contains(fragment, "{value}") {
		return fmt.Errorf("%w: target %s option %s command-line fragment contains template syntax", ErrInvalidTarget, targetID, optionID)
	}
	if containsControl(fragment) {
		return fmt.Errorf("%w: target %s option %s command-line fragment contains control characters", ErrInvalidTarget, targetID, optionID)
	}
	return nil
}

func validateCmdlineTemplate(targetID string, optionID string, template string) error {
	if strings.TrimSpace(template) == "" {
		return fmt.Errorf("%w: target %s option %s command-line template is empty", ErrInvalidTarget, targetID, optionID)
	}
	if strings.TrimSpace(template) != template {
		return fmt.Errorf("%w: target %s option %s command-line template has surrounding whitespace", ErrInvalidTarget, targetID, optionID)
	}
	if strings.Count(template, "{value}") != 1 {
		return fmt.Errorf("%w: target %s option %s command-line template must contain one {value}", ErrInvalidTarget, targetID, optionID)
	}
	if containsControl(template) {
		return fmt.Errorf("%w: target %s option %s command-line template contains control characters", ErrInvalidTarget, targetID, optionID)
	}
	return nil
}

func validOptionID(id string) bool {
	for index, r := range id {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9' && index > 0:
		case (r == '-' || r == '_' || r == '.') && index > 0:
		default:
			return false
		}
	}
	return true
}

func containsSpaceOrControl(value string) bool {
	return strings.ContainsFunc(value, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsControl(r)
	})
}

func containsControl(value string) bool {
	return strings.ContainsFunc(value, unicode.IsControl)
}

func appendCmdline(base string, extra string) string {
	base = strings.TrimSpace(base)
	extra = strings.TrimSpace(extra)
	switch {
	case base == "":
		return extra
	case extra == "":
		return base
	default:
		return base + " " + extra
	}
}
