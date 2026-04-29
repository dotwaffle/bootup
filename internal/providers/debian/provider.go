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
	"strings"

	"github.com/dotwaffle/bootup/internal/provider"
	"github.com/dotwaffle/bootup/verify"
)

const (
	defaultMirrorURL = "https://deb.debian.org/debian"

	targetID   = "debian-trixie-amd64-netboot"
	providerID = "debian"
)

// Config configures the Debian provider.
type Config struct {
	MirrorURL string
	Client    *http.Client
	Keyring   []byte
}

// Provider exposes Debian netboot targets.
type Provider struct {
	mirrorURL string
	client    *http.Client
	keyring   []byte
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
	return &Provider{
		mirrorURL: mirrorURL,
		client:    config.Client,
		keyring:   bytes.Clone(config.Keyring),
	}
}

// ID returns the provider ID.
func (*Provider) ID() string {
	return providerID
}

// Targets returns supported Debian targets.
func (*Provider) Targets(context.Context) ([]provider.Target, error) {
	return []provider.Target{{
		ID:           targetID,
		ProviderID:   providerID,
		Name:         "Debian trixie amd64 netboot",
		Architecture: "amd64",
		Distribution: "debian",
		Release:      "trixie",
		Kind:         "installer",
	}}, nil
}

// Plan returns a boot plan for target.
func (p *Provider) Plan(_ context.Context, target provider.Target) (provider.BootPlan, error) {
	if target.ID != targetID {
		return provider.BootPlan{}, fmt.Errorf("unsupported Debian target %q", target.ID)
	}

	imagesBase := p.mirrorURL + "/dists/trixie/main/installer-amd64/current/images"
	installerBase := imagesBase + "/netboot"
	return provider.BootPlan{
		Target: target,
		Kernel: provider.Artifact{
			Name: "linux",
			URL:  installerBase + "/debian-installer/amd64/linux",
		},
		Initrd: provider.Artifact{
			Name: "initrd.gz",
			URL:  installerBase + "/debian-installer/amd64/initrd.gz",
		},
		Cmdline: "priority=low console=ttyS0",
		Verification: provider.Verification{
			MetadataURL: p.mirrorURL + "/dists/trixie/InRelease",
			ChecksumURL: imagesBase + "/SHA256SUMS",
		},
	}, nil
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
	checksumPath, err := releasePath(config.Plan.Verification.ChecksumURL)
	if err != nil {
		return provider.BootPlan{}, err
	}
	if err := verifyReleaseFileChecksum(checksumPath, shaSums, release); err != nil {
		return provider.BootPlan{}, err
	}
	plan := config.Plan
	if plan.Kernel.Path, err = fetchStageVerify(ctx, client, config.StagingDir, plan.Kernel.URL, "linux", "netboot/debian-installer/amd64/linux", shaSums); err != nil {
		return provider.BootPlan{}, err
	}
	if plan.Initrd.Path, err = fetchStageVerify(ctx, client, config.StagingDir, plan.Initrd.URL, "initrd.gz", "netboot/debian-installer/amd64/initrd.gz", shaSums); err != nil {
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

func releasePath(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse checksum URL: %w", err)
	}
	const marker = "/dists/trixie/"
	index := strings.Index(parsed.Path, marker)
	if index < 0 {
		return "", fmt.Errorf("checksum URL %q does not contain %s", rawURL, marker)
	}
	return parsed.Path[index+len(marker):], nil
}
