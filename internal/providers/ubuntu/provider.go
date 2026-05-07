// Package ubuntu provides Ubuntu netboot targets.
package ubuntu

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
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
	defaultReleaseURL       = "https://releases.ubuntu.com/26.04"
	defaultDiscoveryURL     = "https://releases.ubuntu.com/releases"
	defaultDiscoveryTimeout = 5 * time.Second

	targetID        = "ubuntu-2604-amd64-netboot"
	providerID      = "ubuntu"
	liveServerISO   = "ubuntu-26.04-live-server-amd64.iso"
	kernelStageName = "linux"
	initrdStageName = "initrd"
)

var (
	hrefPattern          = regexp.MustCompile(`(?i)href\s*=\s*["']?([^"'\s>]+)`)
	releaseDirPattern    = regexp.MustCompile(`^\d+\.\d+$`)
	liveServerISOPattern = regexp.MustCompile(`^ubuntu-(\d+\.\d+(?:\.\d+)?)-live-server-amd64\.iso$`)
)

// Config configures the Ubuntu provider.
type Config struct {
	ReleaseURL       string
	DiscoveryURL     string
	DiscoveryFile    string
	Client           *http.Client
	Keyring          []byte
	KernelSHA256     string
	InitrdSHA256     string
	Targets          []provider.Target
	DiscoveryTimeout time.Duration
	Lifecycle        map[string]provider.LifecycleEntry
}

// Provider exposes Ubuntu netboot targets.
type Provider struct {
	releaseURL       string
	discoveryURL     string
	discoveryFile    string
	client           *http.Client
	keyring          []byte
	kernelSHA256     string
	initrdSHA256     string
	targets          []provider.Target
	discoveryTimeout time.Duration
	lifecycle        map[string]provider.LifecycleEntry
}

// NewProvider creates an Ubuntu provider.
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
		releaseURL:       releaseURL,
		discoveryURL:     discoveryURL,
		discoveryFile:    strings.TrimSpace(config.DiscoveryFile),
		client:           config.Client,
		keyring:          bytes.Clone(config.Keyring),
		kernelSHA256:     config.KernelSHA256,
		initrdSHA256:     config.InitrdSHA256,
		targets:          targets,
		discoveryTimeout: discoveryTimeout,
		lifecycle:        cloneLifecycle(config.Lifecycle),
	}
}

// ID returns the provider ID.
func (*Provider) ID() string {
	return providerID
}

// Targets returns supported Ubuntu targets.
func (p *Provider) Targets(context.Context) ([]provider.Target, error) {
	return cloneTargets(p.targets), nil
}

// DiscoveryFamily returns the Ubuntu dynamic discovery family.
func (*Provider) DiscoveryFamily() provider.DiscoveryFamily {
	return provider.DiscoveryFamily{
		ID:          providerID,
		ProviderID:  providerID,
		Name:        "Ubuntu",
		Description: "Discover Ubuntu amd64 netboot installers from the configured releases index.",
	}
}

// DiscoverTargets discovers Ubuntu amd64 netboot targets from the configured
// releases index.
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
	releaseURLs, err := discoverReleaseURLs(ctx, releaseDiscoveryInput{
		Client:               client,
		MetadataDiscoveryURL: p.discoveryMetadataURL(),
		SourceDiscoveryURL:   p.discoveryURL,
	})
	if err != nil {
		return nil, err
	}

	targets := make([]provider.Target, 0, len(releaseURLs))
	for _, releaseURL := range releaseURLs {
		target, ok, err := p.discoverReleaseTarget(ctx, client, releaseURL)
		if err != nil {
			if isContextError(err) {
				return nil, err
			}
			continue
		}
		if ok {
			targets = append(targets, target)
		}
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
		Name:       "Ubuntu 26.04 amd64 netboot",
		Catalog: provider.CatalogEntry{
			Architecture: "amd64",
			Distribution: "ubuntu",
			Release:      "26.04",
			Kind:         "installer",
		},
	}}
}

func cloneTargets(targets []provider.Target) []provider.Target {
	return append([]provider.Target(nil), targets...)
}

func cloneLifecycle(lifecycle map[string]provider.LifecycleEntry) map[string]provider.LifecycleEntry {
	if len(lifecycle) == 0 {
		return nil
	}
	return maps.Clone(lifecycle)
}

// Plan returns a boot plan for target.
func (p *Provider) Plan(_ context.Context, input provider.PlanInput) (provider.BootPlan, error) {
	target := input.Target
	selected, err := p.selectedTarget(target)
	if err != nil {
		return provider.BootPlan{}, err
	}
	architecture := selected.Catalog.Architecture
	if architecture != "amd64" {
		return provider.BootPlan{}, fmt.Errorf("unsupported Ubuntu architecture %q for target %s", architecture, selected.ID)
	}
	releaseURL := targetBaseURL(selected, p.releaseURL)
	isoName, err := targetISOName(selected)
	if err != nil {
		return provider.BootPlan{}, err
	}

	plan := provider.BootPlan{
		Target: selected,
		Kernel: provider.Artifact{
			Name:   kernelStageName,
			URL:    fmt.Sprintf("%s/netboot/%s/linux", releaseURL, architecture),
			SHA256: p.kernelSHA256,
		},
		Initrd: provider.Artifact{
			Name:   initrdStageName,
			URL:    fmt.Sprintf("%s/netboot/%s/initrd", releaseURL, architecture),
			SHA256: p.initrdSHA256,
		},
		Cmdline: "url=" + releaseURL + "/" + isoName + " ip=dhcp console=ttyS0",
		Verification: provider.Verification{
			ChecksumURL:  releaseURL + "/SHA256SUMS",
			SignatureURL: releaseURL + "/SHA256SUMS.gpg",
		},
	}
	return provider.ApplySelectedOptions(plan, input.Options)
}

func (p *Provider) selectedTarget(target provider.Target) (provider.Target, error) {
	if selected, ok := p.target(target.ID); ok {
		return selected, nil
	}
	if err := provider.ValidateTarget(providerID, target); err != nil {
		return provider.Target{}, err
	}
	if target.Catalog.Distribution != providerID {
		return provider.Target{}, fmt.Errorf("unsupported Ubuntu distribution %q for target %s", target.Catalog.Distribution, target.ID)
	}
	if target.Catalog.Kind != "installer" {
		return provider.Target{}, fmt.Errorf("unsupported Ubuntu target kind %q for target %s", target.Catalog.Kind, target.ID)
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

func targetBaseURL(target provider.Target, fallback string) string {
	if target.Source.BaseURL != "" {
		return strings.TrimRight(target.Source.BaseURL, "/")
	}
	return fallback
}

func targetISOName(target provider.Target) (string, error) {
	if target.Source.ISOName != "" {
		return target.Source.ISOName, nil
	}
	if target.ID == targetID {
		return liveServerISO, nil
	}
	return "", fmt.Errorf("source ISO name is required for Ubuntu target %s", target.ID)
}

type releaseCandidate struct {
	metadataURL string
	sourceURL   string
}

type releaseDiscoveryInput struct {
	Client               *http.Client
	MetadataDiscoveryURL string
	SourceDiscoveryURL   string
}

func discoverReleaseURLs(ctx context.Context, input releaseDiscoveryInput) ([]releaseCandidate, error) {
	indexURL := providerhttp.EnsureTrailingSlash(input.MetadataDiscoveryURL)
	body, status, err := providerhttp.FetchStatus(ctx, input.Client, indexURL)
	if err != nil {
		return nil, fmt.Errorf("fetch Ubuntu releases index: %w", err)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("fetch Ubuntu releases index: GET %s: %s", indexURL, http.StatusText(status))
	}

	metadataBase, err := url.Parse(indexURL)
	if err != nil {
		return nil, fmt.Errorf("parse Ubuntu releases index URL: %w", err)
	}
	sourceBase, err := url.Parse(providerhttp.EnsureTrailingSlash(input.SourceDiscoveryURL))
	if err != nil {
		return nil, fmt.Errorf("parse Ubuntu releases source URL: %w", err)
	}
	seen := make(map[string]releaseCandidate)
	for _, match := range hrefPattern.FindAllSubmatch(body, -1) {
		if len(match) < 2 {
			continue
		}
		href := string(match[1])
		parsed, err := url.Parse(href)
		if err != nil {
			continue
		}
		metadataResolved := metadataBase.ResolveReference(parsed)
		if !releaseDirPattern.MatchString(providerhttp.PathBase(metadataResolved.Path)) {
			continue
		}
		sourceResolved := sourceBase.ResolveReference(parsed)
		sourceURL := strings.TrimRight(sourceResolved.String(), "/")
		seen[sourceURL] = releaseCandidate{
			metadataURL: strings.TrimRight(metadataResolved.String(), "/"),
			sourceURL:   sourceURL,
		}
	}
	releaseURLs := make([]releaseCandidate, 0, len(seen))
	for _, releaseURL := range seen {
		releaseURLs = append(releaseURLs, releaseURL)
	}
	sort.Slice(releaseURLs, func(i, j int) bool {
		return releaseURLs[i].sourceURL < releaseURLs[j].sourceURL
	})
	return releaseURLs, nil
}

func (p *Provider) discoverReleaseTarget(ctx context.Context, client *http.Client, release releaseCandidate) (provider.Target, bool, error) {
	shaSums, status, err := providerhttp.FetchStatus(ctx, client, release.metadataURL+"/SHA256SUMS")
	if err != nil {
		return provider.Target{}, false, fmt.Errorf("fetch Ubuntu release metadata for %s: %w", release.metadataURL, err)
	}
	if status == http.StatusNotFound {
		return provider.Target{}, false, nil
	}
	if status != http.StatusOK {
		return provider.Target{}, false, fmt.Errorf("fetch Ubuntu release metadata for %s: %s", release.metadataURL, http.StatusText(status))
	}
	isoName, releaseVersion, ok, err := parseLiveServerISO(shaSums)
	if err != nil {
		return provider.Target{}, false, err
	}
	if !ok {
		return provider.Target{}, false, nil
	}
	if ok, err := providerhttp.Probe(ctx, client, release.metadataURL+"/netboot/amd64/linux"); err != nil || !ok {
		return provider.Target{}, false, err
	}
	if ok, err := providerhttp.Probe(ctx, client, release.metadataURL+"/netboot/amd64/initrd"); err != nil || !ok {
		return provider.Target{}, false, err
	}
	return p.discoveredTarget(release.sourceURL, releaseVersion, isoName), true, nil
}

func parseLiveServerISO(shaSums []byte) (string, string, bool, error) {
	checksums, err := verify.ParseSHA256Sums(bytes.NewReader(shaSums))
	if err != nil {
		return "", "", false, err
	}
	for name := range checksums {
		match := liveServerISOPattern.FindStringSubmatch(name)
		if match == nil {
			continue
		}
		return name, match[1], true, nil
	}
	return "", "", false, nil
}

func (p *Provider) discoveredTarget(releaseURL string, release string, isoName string) provider.Target {
	return provider.Target{
		ID:         "ubuntu-" + strings.ReplaceAll(release, ".", "") + "-amd64-netboot",
		ProviderID: providerID,
		Name:       "Ubuntu " + release + " amd64 netboot",
		Catalog: provider.CatalogEntry{
			Distribution: providerID,
			Release:      release,
			Architecture: "amd64",
			Kind:         "installer",
		},
		Source: provider.SourceEntry{
			BaseURL: releaseURL,
			ISOName: isoName,
		},
		Lifecycle: p.lifecycleEntry(release),
	}
}

func (p *Provider) lifecycleEntry(release string) provider.LifecycleEntry {
	if entry, ok := p.lifecycle[release]; ok {
		return entry
	}
	return provider.LifecycleEntry{
		Status: provider.LifecycleUnknown,
		Source: "ubuntu",
	}
}

func isContextError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// Stage downloads, verifies, and stages artifacts for plan.
func (p *Provider) Stage(ctx context.Context, config provider.StageConfig) (provider.BootPlan, error) {
	var keyring io.Reader
	if len(p.keyring) > 0 {
		keyring = bytes.NewReader(p.keyring)
	}
	return FetchAndStageArtifacts(ctx, FetchConfig{
		Plan:       config.Plan,
		Client:     p.client,
		Keyring:    keyring,
		StagingDir: config.StagingDir,
		Progress:   config.Progress,
	})
}

// FetchConfig configures Ubuntu artifact fetching and staging.
type FetchConfig struct {
	Plan       provider.BootPlan
	Client     *http.Client
	Keyring    io.Reader
	StagingDir string
	Progress   provider.StageProgressFunc
}

// FetchAndStageArtifacts downloads Ubuntu metadata and artifacts, verifies the
// configured trust chain, and stages verified artifacts on disk.
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

	if config.Keyring != nil {
		shaSums, err := fetchWithProgress(ctx, client, plan.Verification.ChecksumURL, "SHA256SUMS", config.Progress)
		if err != nil {
			return provider.BootPlan{}, fmt.Errorf("fetch SHA256SUMS: %w", err)
		}
		signature, err := fetchWithProgress(ctx, client, plan.Verification.SignatureURL, "SHA256SUMS.gpg", config.Progress)
		if err != nil {
			return provider.BootPlan{}, fmt.Errorf("fetch SHA256SUMS.gpg: %w", err)
		}
		if err := reportProgress(config.Progress, provider.StageOperationVerify, provider.StageStateStarted, "SHA256SUMS"); err != nil {
			return provider.BootPlan{}, err
		}
		if err := verify.Artifact(verify.ArtifactInput{
			Artifact:  bytes.NewReader(shaSums),
			Signature: bytes.NewReader(signature),
			Keyring:   config.Keyring,
			Name:      "SHA256SUMS",
		}); err != nil {
			return provider.BootPlan{}, err
		}
		if err := reportProgress(config.Progress, provider.StageOperationVerify, provider.StageStateCompleted, "SHA256SUMS"); err != nil {
			return provider.BootPlan{}, err
		}
		isoName, err := targetISOName(plan.Target)
		if err != nil {
			return provider.BootPlan{}, err
		}
		if err := requireChecksumEntry(shaSums, isoName); err != nil {
			return provider.BootPlan{}, err
		}
	}
	if plan.Kernel.SHA256 == "" {
		if err := requireHTTPS(plan.Kernel.URL, plan.Initrd.URL); err != nil {
			return provider.BootPlan{}, err
		}
	}

	var err error
	if plan.Kernel.Path, err = fetchStageVerify(ctx, client, config.StagingDir, plan.Kernel, config.Progress); err != nil {
		return provider.BootPlan{}, err
	}
	if plan.Initrd.Path, err = fetchStageVerify(ctx, client, config.StagingDir, plan.Initrd, config.Progress); err != nil {
		return provider.BootPlan{}, err
	}
	return plan, nil
}

func requireChecksumEntry(shaSums []byte, name string) error {
	checksums, err := verify.ParseSHA256Sums(bytes.NewReader(shaSums))
	if err != nil {
		return err
	}
	if _, ok := checksums[name]; !ok {
		return fmt.Errorf("checksum for %q not found", name)
	}
	return nil
}

func requireHTTPS(rawURLs ...string) error {
	for _, rawURL := range rawURLs {
		parsed, err := url.Parse(rawURL)
		if err != nil {
			return fmt.Errorf("parse artifact URL: %w", err)
		}
		if parsed.Scheme != "https" {
			return fmt.Errorf("unverified Ubuntu artifact URL must use https: %s", rawURL)
		}
	}
	return nil
}

func fetchStageVerify(ctx context.Context, client *http.Client, dir string, artifact provider.Artifact, progress provider.StageProgressFunc) (string, error) {
	data, err := fetchWithProgress(ctx, client, artifact.URL, artifact.Name, progress)
	if err != nil {
		return "", fmt.Errorf("fetch %s: %w", artifact.Name, err)
	}
	if artifact.SHA256 != "" {
		if err := reportProgress(progress, provider.StageOperationVerify, provider.StageStateStarted, artifact.Name); err != nil {
			return "", err
		}
		if err := verify.SHA256(verify.HashInput{
			Artifact:       bytes.NewReader(data),
			ExpectedSHA256: artifact.SHA256,
			Name:           artifact.Name,
		}); err != nil {
			return "", err
		}
		if err := reportProgress(progress, provider.StageOperationVerify, provider.StageStateCompleted, artifact.Name); err != nil {
			return "", err
		}
	}
	if err := reportProgress(progress, provider.StageOperationWrite, provider.StageStateStarted, artifact.Name); err != nil {
		return "", err
	}
	path := filepath.Join(dir, artifact.Name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("stage %s: %w", artifact.Name, err)
	}
	if err := reportProgress(progress, provider.StageOperationWrite, provider.StageStateCompleted, artifact.Name); err != nil {
		return "", err
	}
	return path, nil
}

func fetchWithProgress(ctx context.Context, client *http.Client, rawURL string, artifact string, progress provider.StageProgressFunc) ([]byte, error) {
	if err := reportProgress(progress, provider.StageOperationFetch, provider.StageStateStarted, artifact); err != nil {
		return nil, err
	}
	data, err := fetch(ctx, client, rawURL)
	if err != nil {
		return nil, err
	}
	if err := reportProgress(progress, provider.StageOperationFetch, provider.StageStateCompleted, artifact); err != nil {
		return nil, err
	}
	return data, nil
}

func reportProgress(progress provider.StageProgressFunc, operation provider.StageOperation, state provider.StageState, artifact string) error {
	return provider.ReportStageProgress(progress, provider.StageProgress{
		Operation: operation,
		State:     state,
		Artifact:  artifact,
	})
}

func fetch(ctx context.Context, client *http.Client, rawURL string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", rawURL, response.Status)
	}
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	return data, nil
}
