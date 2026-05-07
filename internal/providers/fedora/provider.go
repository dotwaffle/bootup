// Package fedora provides Fedora Server netboot targets.
package fedora

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/dotwaffle/bootup/internal/provider"
	"github.com/dotwaffle/bootup/internal/providerhttp"
	"github.com/dotwaffle/bootup/verify"
)

const (
	defaultReleaseURL       = "https://download.fedoraproject.org/pub/fedora/linux/releases/44/Server/x86_64/os"
	defaultDiscoveryURL     = "https://download.fedoraproject.org/pub/fedora/linux/releases"
	defaultDiscoveryTimeout = 5 * time.Second

	targetID        = "fedora-44-amd64-server-netboot"
	providerID      = "fedora"
	kernelStageName = "vmlinuz"
	initrdStageName = "initrd.img"
)

var (
	hrefPattern       = regexp.MustCompile(`(?i)href\s*=\s*["']?([^"'\s>]+)`)
	releaseDirPattern = regexp.MustCompile(`^\d+$`)
)

// Config configures the Fedora provider.
type Config struct {
	ReleaseURL       string
	DiscoveryURL     string
	DiscoveryFile    string
	Client           *http.Client
	KernelSHA256     string
	InitrdSHA256     string
	Targets          []provider.Target
	DiscoveryTimeout time.Duration
}

// Provider exposes Fedora Server netboot targets.
type Provider struct {
	releaseURL           string
	discoveryURL         string
	discoveryFile        string
	client               *http.Client
	kernelSHA256         string
	initrdSHA256         string
	targets              []provider.Target
	discoveryTimeout     time.Duration
	releaseURLConfigured bool
}

// NewProvider creates a Fedora provider.
func NewProvider(config Config) *Provider {
	releaseURL := strings.TrimRight(config.ReleaseURL, "/")
	if releaseURL == "" {
		releaseURL = defaultReleaseURL
	}
	discoveryURL := strings.TrimRight(config.DiscoveryURL, "/")
	if discoveryURL == "" {
		discoveryURL = defaultDiscoveryURL
	}
	targets := cloneTargets(config.Targets)
	if config.Targets == nil {
		targets = defaultTargets()
	}
	discoveryTimeout := config.DiscoveryTimeout
	if discoveryTimeout <= 0 {
		discoveryTimeout = defaultDiscoveryTimeout
	}
	return &Provider{
		releaseURL:           releaseURL,
		discoveryURL:         discoveryURL,
		discoveryFile:        strings.TrimSpace(config.DiscoveryFile),
		client:               config.Client,
		kernelSHA256:         config.KernelSHA256,
		initrdSHA256:         config.InitrdSHA256,
		targets:              targets,
		discoveryTimeout:     discoveryTimeout,
		releaseURLConfigured: strings.TrimSpace(config.ReleaseURL) != "",
	}
}

// ID returns the provider ID.
func (*Provider) ID() string {
	return providerID
}

// Targets returns supported Fedora targets.
func (p *Provider) Targets(context.Context) ([]provider.Target, error) {
	return cloneTargets(p.targets), nil
}

// DiscoveryFamily returns the Fedora dynamic discovery family.
func (*Provider) DiscoveryFamily() provider.DiscoveryFamily {
	return provider.DiscoveryFamily{
		ID:          providerID,
		ProviderID:  providerID,
		Name:        "Fedora",
		Description: "Discover Fedora Server amd64 netboot installers from the configured releases index.",
	}
}

// DiscoverTargets discovers Fedora Server amd64 netboot targets from the
// configured releases index.
func (p *Provider) DiscoverTargets(ctx context.Context) ([]provider.Target, error) {
	if p.discoveryTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.discoveryTimeout)
		defer cancel()
	}

	client := p.client
	if client == nil {
		client = http.DefaultClient
	}
	indexURL := providerhttp.EnsureTrailingSlash(p.discoveryMetadataURL())
	sourceIndexURL := providerhttp.EnsureTrailingSlash(p.discoveryURL)
	body, status, err := providerhttp.FetchStatus(ctx, client, indexURL)
	if err != nil {
		return nil, fmt.Errorf("fetch Fedora releases index: %w", err)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("fetch Fedora releases index: GET %s: %s", indexURL, http.StatusText(status))
	}

	releases := parseReleasesIndex(body)
	targets := make([]provider.Target, 0, len(releases))
	for _, release := range releases {
		metadataBaseURL := indexURL + release + "/Server/x86_64/os"
		sourceBaseURL := sourceIndexURL + release + "/Server/x86_64/os"
		kernelOK, err := providerhttp.Probe(ctx, client, metadataBaseURL+"/images/pxeboot/vmlinuz")
		if err != nil {
			if isContextError(err) {
				return nil, fmt.Errorf("probe Fedora %s kernel: %w", release, err)
			}
			continue
		}
		if !kernelOK {
			continue
		}
		initrdOK, err := providerhttp.Probe(ctx, client, metadataBaseURL+"/images/pxeboot/initrd.img")
		if err != nil {
			if isContextError(err) {
				return nil, fmt.Errorf("probe Fedora %s initrd: %w", release, err)
			}
			continue
		}
		if !initrdOK {
			continue
		}
		targets = append(targets, discoveredTarget(release, sourceBaseURL))
	}
	return targets, nil
}

func (p *Provider) discoveryMetadataURL() string {
	if p.discoveryFile != "" {
		return providerhttp.LocalFileURL(p.discoveryFile)
	}
	return p.discoveryURL
}

func defaultTargets() []provider.Target {
	return []provider.Target{{
		ID:         targetID,
		ProviderID: providerID,
		Name:       "Fedora Server 44 amd64 netboot",
		Catalog: provider.CatalogEntry{
			Architecture: "amd64",
			Distribution: providerID,
			Release:      "44",
			Kind:         "installer",
		},
		Source: provider.SourceEntry{
			BaseURL: defaultReleaseURL,
		},
		Lifecycle: provider.LifecycleEntry{
			Status: provider.LifecycleSupported,
			Source: "catalog",
		},
	}}
}

func cloneTargets(targets []provider.Target) []provider.Target {
	return append([]provider.Target(nil), targets...)
}

func parseReleasesIndex(data []byte) []string {
	seen := make(map[string]struct{})
	for _, match := range hrefPattern.FindAllSubmatch(data, -1) {
		if len(match) < 2 {
			continue
		}
		release := strings.TrimSuffix(string(match[1]), "/")
		release = providerhttp.PathBase(release)
		if releaseDirPattern.MatchString(release) {
			seen[release] = struct{}{}
		}
	}
	releases := make([]string, 0, len(seen))
	for release := range seen {
		releases = append(releases, release)
	}
	sort.Strings(releases)
	return releases
}

func discoveredTarget(release string, baseURL string) provider.Target {
	return provider.Target{
		ID:         "fedora-" + release + "-amd64-server-netboot",
		ProviderID: providerID,
		Name:       "Fedora Server " + release + " amd64 netboot",
		Catalog: provider.CatalogEntry{
			Architecture: "amd64",
			Distribution: providerID,
			Release:      release,
			Kind:         "installer",
		},
		Source: provider.SourceEntry{
			BaseURL: baseURL,
		},
	}
}

func isContextError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// Plan returns a boot plan for target.
func (p *Provider) Plan(ctx context.Context, input provider.PlanInput) (provider.BootPlan, error) {
	target := input.Target
	selected, err := p.selectedTarget(target)
	if err != nil {
		return provider.BootPlan{}, err
	}
	architecture := selected.Catalog.Architecture
	if architecture != "amd64" {
		return provider.BootPlan{}, fmt.Errorf("unsupported Fedora architecture %q for target %s", architecture, selected.ID)
	}
	releaseURL := p.targetBaseURL(selected)
	checksums, err := p.artifactChecksums(ctx, selected, releaseURL, input.Offline)
	if err != nil {
		return provider.BootPlan{}, err
	}

	plan := provider.BootPlan{
		Target: selected,
		Kernel: provider.Artifact{
			Name:   kernelStageName,
			URL:    releaseURL + "/" + kernelTreeinfoPath,
			SHA256: checksums.kernelSHA256,
		},
		Initrd: provider.Artifact{
			Name:   initrdStageName,
			URL:    releaseURL + "/" + initrdTreeinfoPath,
			SHA256: checksums.initrdSHA256,
		},
		Cmdline: "inst.repo=" + releaseURL + " ip=dhcp console=ttyS0",
	}
	return provider.ApplySelectedOptions(plan, input.Options)
}

func (p *Provider) artifactChecksums(ctx context.Context, target provider.Target, releaseURL string, offline bool) (treeinfoChecksums, error) {
	if (p.kernelSHA256 == "") != (p.initrdSHA256 == "") {
		return treeinfoChecksums{}, errors.New("kernel and initrd sha256 must be supplied together")
	}
	if p.kernelSHA256 != "" {
		return treeinfoChecksums{
			kernelSHA256: p.kernelSHA256,
			initrdSHA256: p.initrdSHA256,
		}, nil
	}
	if (target.Source.KernelSHA256 == "") != (target.Source.InitrdSHA256 == "") {
		return treeinfoChecksums{}, fmt.Errorf("target %s source kernel and initrd sha256 must be supplied together", target.ID)
	}
	if target.Source.KernelSHA256 != "" {
		return treeinfoChecksums{
			kernelSHA256: strings.ToLower(target.Source.KernelSHA256),
			initrdSHA256: strings.ToLower(target.Source.InitrdSHA256),
		}, nil
	}
	if offline {
		return treeinfoChecksums{}, fmt.Errorf("fedora .treeinfo checksums are required for target %s but offline planning cannot fetch remote metadata", target.ID)
	}
	treeinfoURL := releaseURL + "/.treeinfo"
	if err := requireHTTPS(treeinfoURL); err != nil {
		return treeinfoChecksums{}, err
	}
	client := p.client
	if client == nil {
		client = http.DefaultClient
	}
	data, err := providerhttp.Fetch(ctx, client, treeinfoURL)
	if err != nil {
		return treeinfoChecksums{}, fmt.Errorf("fetch Fedora .treeinfo: %w", err)
	}
	checksums, err := parseTreeinfoChecksums(data)
	if err != nil {
		return treeinfoChecksums{}, fmt.Errorf("parse Fedora .treeinfo: %w", err)
	}
	return checksums, nil
}

func (p *Provider) selectedTarget(target provider.Target) (provider.Target, error) {
	if selected, ok := p.target(target.ID); ok {
		return selected, nil
	}
	if err := provider.ValidateTarget(providerID, target); err != nil {
		return provider.Target{}, err
	}
	if target.Catalog.Distribution != providerID {
		return provider.Target{}, fmt.Errorf("unsupported Fedora distribution %q for target %s", target.Catalog.Distribution, target.ID)
	}
	if target.Catalog.Kind != "installer" {
		return provider.Target{}, fmt.Errorf("unsupported Fedora target kind %q for target %s", target.Catalog.Kind, target.ID)
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

func (p *Provider) targetBaseURL(target provider.Target) string {
	if target.Source.BaseURL != "" && !p.releaseURLConfigured {
		return strings.TrimRight(target.Source.BaseURL, "/")
	}
	return p.releaseURL
}

// Stage downloads, verifies, and stages artifacts for plan.
func (p *Provider) Stage(ctx context.Context, config provider.StageConfig) (provider.BootPlan, error) {
	return FetchAndStageArtifacts(ctx, FetchConfig{
		Plan:       config.Plan,
		Client:     p.client,
		StagingDir: config.StagingDir,
	})
}

// FetchConfig configures Fedora artifact fetching and staging.
type FetchConfig struct {
	Plan       provider.BootPlan
	Client     *http.Client
	StagingDir string
}

// FetchAndStageArtifacts downloads Fedora artifacts and stages them on disk.
func FetchAndStageArtifacts(ctx context.Context, config FetchConfig) (provider.BootPlan, error) {
	client := config.Client
	if client == nil {
		client = http.DefaultClient
	}
	if config.StagingDir == "" {
		return provider.BootPlan{}, errors.New("staging dir is required")
	}

	plan := config.Plan
	if (plan.Kernel.SHA256 == "") != (plan.Initrd.SHA256 == "") {
		return provider.BootPlan{}, errors.New("kernel and initrd sha256 must be supplied together")
	}
	if plan.Kernel.SHA256 == "" {
		if err := requireHTTPS(plan.Kernel.URL, plan.Initrd.URL); err != nil {
			return provider.BootPlan{}, err
		}
	}

	var err error
	if plan.Kernel.Path, err = fetchStageVerify(ctx, client, config.StagingDir, plan.Kernel); err != nil {
		return provider.BootPlan{}, err
	}
	if plan.Initrd.Path, err = fetchStageVerify(ctx, client, config.StagingDir, plan.Initrd); err != nil {
		return provider.BootPlan{}, err
	}
	return plan, nil
}

func requireHTTPS(rawURLs ...string) error {
	for _, rawURL := range rawURLs {
		parsed, err := url.Parse(rawURL)
		if err != nil {
			return fmt.Errorf("parse artifact URL: %w", err)
		}
		if parsed.Scheme != "https" {
			return fmt.Errorf("unverified Fedora artifact URL must use https: %s", rawURL)
		}
	}
	return nil
}

func fetchStageVerify(ctx context.Context, client *http.Client, dir string, artifact provider.Artifact) (string, error) {
	data, err := providerhttp.Fetch(ctx, client, artifact.URL)
	if err != nil {
		return "", fmt.Errorf("fetch %s: %w", artifact.Name, err)
	}
	if artifact.SHA256 != "" {
		if err := verify.SHA256(verify.HashInput{
			Artifact:       bytes.NewReader(data),
			ExpectedSHA256: artifact.SHA256,
			Name:           artifact.Name,
		}); err != nil {
			return "", err
		}
	}
	path := filepath.Join(dir, artifact.Name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("stage %s: %w", artifact.Name, err)
	}
	return path, nil
}
