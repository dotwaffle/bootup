// Package debian provides Debian netboot targets.
package debian

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/dotwaffle/bootup/internal/provider"
	"github.com/dotwaffle/bootup/verify"
)

const (
	defaultMirrorURL        = "https://deb.debian.org/debian"
	defaultDiscoveryTimeout = 5 * time.Second

	targetID   = "debian-trixie-amd64-netboot"
	providerID = "debian"
)

var hrefPattern = regexp.MustCompile(`(?i)href\s*=\s*["']?([^"'\s>]+)`)

// Config configures the Debian provider.
type Config struct {
	MirrorURL        string
	Client           *http.Client
	Keyring          []byte
	Targets          []provider.Target
	DiscoveryTimeout time.Duration
}

// Provider exposes Debian netboot targets.
type Provider struct {
	mirrorURL        string
	client           *http.Client
	keyring          []byte
	targets          []provider.Target
	discoveryTimeout time.Duration
}

// FetchConfig configures Debian artifact fetching and staging.
type FetchConfig struct {
	Plan       provider.BootPlan
	Client     *http.Client
	Keyring    io.Reader
	StagingDir string
}

// NewProvider creates a Debian provider.
func NewProvider(config Config) *Provider {
	mirrorURL := strings.TrimRight(config.MirrorURL, "/")
	if mirrorURL == "" {
		mirrorURL = defaultMirrorURL
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
		mirrorURL:        mirrorURL,
		client:           config.Client,
		keyring:          bytes.Clone(config.Keyring),
		targets:          targets,
		discoveryTimeout: discoveryTimeout,
	}
}

// ID returns the provider ID.
func (*Provider) ID() string {
	return providerID
}

// Targets returns supported Debian targets.
func (p *Provider) Targets(context.Context) ([]provider.Target, error) {
	return cloneTargets(p.targets), nil
}

// DiscoveryFamily returns the Debian dynamic discovery family.
func (*Provider) DiscoveryFamily() provider.DiscoveryFamily {
	return provider.DiscoveryFamily{
		ID:          providerID,
		ProviderID:  providerID,
		Name:        "Debian",
		Description: "Discover Debian amd64 netboot installers from the configured mirror.",
	}
}

// DiscoverTargets discovers Debian amd64 netboot targets from the configured
// mirror.
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
	releases, err := discoverReleases(ctx, client, p.mirrorURL)
	if err != nil {
		return nil, err
	}

	targets := make([]provider.Target, 0, len(releases))
	for _, release := range releases {
		ok, err := hasAMD64Netboot(ctx, client, p.mirrorURL, release)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		targets = append(targets, discoveredTarget(p.mirrorURL, release))
	}
	return targets, nil
}

func defaultTargets() []provider.Target {
	return []provider.Target{{
		ID:         targetID,
		ProviderID: providerID,
		Name:       "Debian trixie amd64 netboot",
		Catalog: provider.CatalogEntry{
			Architecture: "amd64",
			Distribution: "debian",
			Release:      "trixie",
			Kind:         "installer",
		},
	}}
}

func cloneTargets(targets []provider.Target) []provider.Target {
	return append([]provider.Target(nil), targets...)
}

// Plan returns a boot plan for target.
func (p *Provider) Plan(_ context.Context, target provider.Target) (provider.BootPlan, error) {
	selected, err := p.selectedTarget(target)
	if err != nil {
		return provider.BootPlan{}, err
	}
	release := selected.Catalog.Release
	architecture := selected.Catalog.Architecture
	if architecture != "amd64" {
		return provider.BootPlan{}, fmt.Errorf("unsupported Debian architecture %q for target %s", architecture, selected.ID)
	}
	baseURL := targetBaseURL(selected, p.mirrorURL)

	imagesBase := fmt.Sprintf("%s/dists/%s/main/installer-%s/current/images", baseURL, release, architecture)
	installerBase := imagesBase + "/netboot"
	return provider.BootPlan{
		Target: selected,
		Kernel: provider.Artifact{
			Name: "linux",
			URL:  fmt.Sprintf("%s/debian-installer/%s/linux", installerBase, architecture),
		},
		Initrd: provider.Artifact{
			Name: "initrd.gz",
			URL:  fmt.Sprintf("%s/debian-installer/%s/initrd.gz", installerBase, architecture),
		},
		Cmdline: "priority=low console=ttyS0",
		Verification: provider.Verification{
			MetadataURL: fmt.Sprintf("%s/dists/%s/InRelease", baseURL, release),
			ChecksumURL: imagesBase + "/SHA256SUMS",
		},
	}, nil
}

func (p *Provider) selectedTarget(target provider.Target) (provider.Target, error) {
	if selected, ok := p.target(target.ID); ok {
		return selected, nil
	}
	if err := provider.ValidateTarget(providerID, target); err != nil {
		return provider.Target{}, err
	}
	if target.Catalog.Distribution != providerID {
		return provider.Target{}, fmt.Errorf("unsupported Debian distribution %q for target %s", target.Catalog.Distribution, target.ID)
	}
	if target.Catalog.Kind != "installer" {
		return provider.Target{}, fmt.Errorf("unsupported Debian target kind %q for target %s", target.Catalog.Kind, target.ID)
	}
	return target, nil
}

func targetBaseURL(target provider.Target, fallback string) string {
	if target.Source.BaseURL != "" {
		return strings.TrimRight(target.Source.BaseURL, "/")
	}
	return fallback
}

func (p *Provider) target(id string) (provider.Target, bool) {
	for _, target := range p.targets {
		if target.ID == id {
			return target, true
		}
	}
	return provider.Target{}, false
}

func discoverReleases(ctx context.Context, client *http.Client, mirrorURL string) ([]string, error) {
	body, status, err := fetchDiscovery(ctx, client, mirrorURL+"/dists/")
	if err != nil {
		return nil, fmt.Errorf("fetch Debian dists index: %w", err)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("fetch Debian dists index: GET %s/dists/: %s", mirrorURL, http.StatusText(status))
	}
	return parseDistsIndex(body), nil
}

func parseDistsIndex(data []byte) []string {
	seen := make(map[string]struct{})
	for _, match := range hrefPattern.FindAllSubmatch(data, -1) {
		if len(match) < 2 {
			continue
		}
		release := strings.TrimSuffix(string(match[1]), "/")
		release = pathBase(release)
		if !isDiscoveryRelease(release) {
			continue
		}
		seen[release] = struct{}{}
	}
	releases := make([]string, 0, len(seen))
	for release := range seen {
		releases = append(releases, release)
	}
	sort.Strings(releases)
	return releases
}

func pathBase(value string) string {
	value = strings.TrimRight(value, "/")
	if index := strings.LastIndex(value, "/"); index >= 0 {
		return value[index+1:]
	}
	return value
}

func isDiscoveryRelease(release string) bool {
	if release == "" || strings.HasPrefix(release, ".") {
		return false
	}
	switch release {
	case "stable", "oldstable", "oldoldstable", "testing", "unstable", "sid", "experimental", "rc-buggy":
		return false
	}
	for _, r := range release {
		if r >= 'a' && r <= 'z' {
			continue
		}
		if r >= '0' && r <= '9' {
			continue
		}
		if r == '-' {
			continue
		}
		return false
	}
	return true
}

func hasAMD64Netboot(ctx context.Context, client *http.Client, mirrorURL string, release string) (bool, error) {
	checksumURL := fmt.Sprintf("%s/dists/%s/main/installer-amd64/current/images/SHA256SUMS", mirrorURL, release)
	body, status, err := fetchDiscovery(ctx, client, checksumURL)
	if err != nil {
		return false, fmt.Errorf("fetch Debian %s amd64 netboot metadata: %w", release, err)
	}
	if status == http.StatusNotFound {
		return false, nil
	}
	if status != http.StatusOK {
		return false, fmt.Errorf("fetch Debian %s amd64 netboot metadata: GET %s: %s", release, checksumURL, http.StatusText(status))
	}
	return bytes.Contains(body, []byte("netboot/debian-installer/amd64/linux")) &&
		bytes.Contains(body, []byte("netboot/debian-installer/amd64/initrd.gz")), nil
}

func discoveredTarget(mirrorURL string, release string) provider.Target {
	return provider.Target{
		ID:         "debian-" + release + "-amd64-netboot",
		ProviderID: providerID,
		Name:       "Debian " + release + " amd64 netboot",
		Catalog: provider.CatalogEntry{
			Distribution: providerID,
			Release:      release,
			Architecture: "amd64",
			Kind:         "installer",
		},
		Source: provider.SourceEntry{
			BaseURL: mirrorURL,
		},
		Lifecycle: provider.LifecycleEntry{
			Status: provider.LifecycleUnknown,
			Source: "debian",
		},
	}
}

func fetchDiscovery(ctx context.Context, client *http.Client, rawURL string) ([]byte, int, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("new request: %w", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = response.Body.Close() }()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("read response: %w", err)
	}
	return data, response.StatusCode, nil
}

// Stage downloads, verifies, and stages artifacts for plan.
func (p *Provider) Stage(ctx context.Context, config provider.StageConfig) (provider.BootPlan, error) {
	if len(p.keyring) == 0 {
		return provider.BootPlan{}, errors.New("keyring is required")
	}
	return FetchAndStageArtifacts(ctx, FetchConfig{
		Plan:       config.Plan,
		Client:     p.client,
		Keyring:    bytes.NewReader(p.keyring),
		StagingDir: config.StagingDir,
	})
}

// VerifyInRelease verifies a clearsigned Debian InRelease file and returns its
// trusted plaintext.
func VerifyInRelease(inRelease []byte, keyringReader io.Reader) ([]byte, error) {
	return verify.ClearSigned(verify.ClearSignedInput{
		Message: inRelease,
		Keyring: keyringReader,
		Name:    "InRelease",
	})
}

// ParseSHA256Sums parses a Debian SHA256SUMS file.
func ParseSHA256Sums(data []byte) (map[string]string, error) {
	return verify.ParseSHA256Sums(bytes.NewReader(data))
}

// VerifyArtifactChecksum verifies data against a parsed SHA256SUMS entry.
func VerifyArtifactChecksum(name string, data []byte, checksums map[string]string) error {
	want, ok := checksums[name]
	if !ok {
		return fmt.Errorf("checksum for %q not found", name)
	}
	return verify.SHA256(verify.HashInput{
		Artifact:       bytes.NewReader(data),
		ExpectedSHA256: want,
		Name:           name,
	})
}

// FetchAndStageArtifacts downloads Debian metadata and artifacts, verifies the
// trust chain, and stages verified artifacts on disk.
func FetchAndStageArtifacts(ctx context.Context, config FetchConfig) (provider.BootPlan, error) {
	client := config.Client
	if client == nil {
		client = http.DefaultClient
	}
	if config.Keyring == nil {
		return provider.BootPlan{}, errors.New("keyring is required")
	}
	if config.StagingDir == "" {
		return provider.BootPlan{}, errors.New("staging dir is required")
	}

	inRelease, err := fetch(ctx, client, config.Plan.Verification.MetadataURL)
	if err != nil {
		return provider.BootPlan{}, fmt.Errorf("fetch InRelease: %w", err)
	}
	release, err := VerifyInRelease(inRelease, config.Keyring)
	if err != nil {
		return provider.BootPlan{}, err
	}

	shaSums, err := fetch(ctx, client, config.Plan.Verification.ChecksumURL)
	if err != nil {
		return provider.BootPlan{}, fmt.Errorf("fetch SHA256SUMS: %w", err)
	}
	releaseName := config.Plan.Target.Catalog.Release
	checksumPath, err := releasePath(config.Plan.Verification.ChecksumURL, releaseName)
	if err != nil {
		return provider.BootPlan{}, err
	}
	if err := verifyReleaseFileChecksum(checksumPath, shaSums, release); err != nil {
		return provider.BootPlan{}, err
	}
	plan := config.Plan
	architecture := plan.Target.Catalog.Architecture
	kernelChecksumName := fmt.Sprintf("netboot/debian-installer/%s/linux", architecture)
	if plan.Kernel.Path, err = fetchStageVerify(ctx, client, config.StagingDir, plan.Kernel.URL, "linux", kernelChecksumName, shaSums); err != nil {
		return provider.BootPlan{}, err
	}
	initrdChecksumName := fmt.Sprintf("netboot/debian-installer/%s/initrd.gz", architecture)
	if plan.Initrd.Path, err = fetchStageVerify(ctx, client, config.StagingDir, plan.Initrd.URL, "initrd.gz", initrdChecksumName, shaSums); err != nil {
		return provider.BootPlan{}, err
	}
	return plan, nil
}

func fetchStageVerify(ctx context.Context, client *http.Client, dir string, artifactURL string, filename string, checksumName string, shaSums []byte) (string, error) {
	data, err := fetch(ctx, client, artifactURL)
	if err != nil {
		return "", fmt.Errorf("fetch %s: %w", filename, err)
	}
	if err := verify.Artifact(verify.ArtifactInput{
		Artifact:   bytes.NewReader(data),
		Name:       checksumName,
		SHA256Sums: bytes.NewReader(shaSums),
	}); err != nil {
		return "", err
	}
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("stage %s: %w", filename, err)
	}
	return path, nil
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

func verifyReleaseFileChecksum(name string, data []byte, release []byte) error {
	checksums, err := parseReleaseSHA256(release)
	if err != nil {
		return err
	}
	return VerifyArtifactChecksum(name, data, checksums)
}

func parseReleaseSHA256(release []byte) (map[string]string, error) {
	checksums := make(map[string]string)
	inSHA256 := false
	for line := range strings.SplitSeq(string(release), "\n") {
		if strings.HasSuffix(line, ":") {
			inSHA256 = line == "SHA256:"
			continue
		}
		if !inSHA256 || strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			return nil, fmt.Errorf("parse Release SHA256 line %q", line)
		}
		checksums[fields[2]] = fields[0]
	}
	return checksums, nil
}

func releasePath(rawURL string, release string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse checksum URL: %w", err)
	}
	marker := fmt.Sprintf("/dists/%s/", release)
	index := strings.Index(parsed.Path, marker)
	if index < 0 {
		return "", fmt.Errorf("checksum URL %q does not contain %s", rawURL, marker)
	}
	return parsed.Path[index+len(marker):], nil
}
