// Package ubuntu provides Ubuntu netboot targets.
package ubuntu

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
	defaultReleaseURL = "https://releases.ubuntu.com/26.04"

	targetID        = "ubuntu-2604-amd64-netboot"
	providerID      = "ubuntu"
	liveServerISO   = "ubuntu-26.04-live-server-amd64.iso"
	kernelStageName = "linux"
	initrdStageName = "initrd"
)

// Config configures the Ubuntu provider.
type Config struct {
	ReleaseURL   string
	Client       *http.Client
	Keyring      []byte
	KernelSHA256 string
	InitrdSHA256 string
}

// Provider exposes Ubuntu netboot targets.
type Provider struct {
	releaseURL   string
	client       *http.Client
	keyring      []byte
	kernelSHA256 string
	initrdSHA256 string
}

// NewProvider creates an Ubuntu provider.
func NewProvider(config Config) *Provider {
	releaseURL := strings.TrimRight(config.ReleaseURL, "/")
	if releaseURL == "" {
		releaseURL = defaultReleaseURL
	}
	return &Provider{
		releaseURL:   releaseURL,
		client:       config.Client,
		keyring:      bytes.Clone(config.Keyring),
		kernelSHA256: config.KernelSHA256,
		initrdSHA256: config.InitrdSHA256,
	}
}

// ID returns the provider ID.
func (*Provider) ID() string {
	return providerID
}

// Targets returns supported Ubuntu targets.
func (*Provider) Targets(context.Context) ([]provider.Target, error) {
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
	}}, nil
}

// Plan returns a boot plan for target.
func (p *Provider) Plan(_ context.Context, target provider.Target) (provider.BootPlan, error) {
	if target.ID != targetID {
		return provider.BootPlan{}, fmt.Errorf("unsupported Ubuntu target %q", target.ID)
	}

	return provider.BootPlan{
		Target: target,
		Kernel: provider.Artifact{
			Name:   kernelStageName,
			URL:    p.releaseURL + "/netboot/amd64/linux",
			SHA256: p.kernelSHA256,
		},
		Initrd: provider.Artifact{
			Name:   initrdStageName,
			URL:    p.releaseURL + "/netboot/amd64/initrd",
			SHA256: p.initrdSHA256,
		},
		Cmdline: "url=" + p.releaseURL + "/" + liveServerISO + " ip=dhcp console=ttyS0",
		Verification: provider.Verification{
			ChecksumURL:  p.releaseURL + "/SHA256SUMS",
			SignatureURL: p.releaseURL + "/SHA256SUMS.gpg",
		},
	}, nil
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
	})
}

// FetchConfig configures Ubuntu artifact fetching and staging.
type FetchConfig struct {
	Plan       provider.BootPlan
	Client     *http.Client
	Keyring    io.Reader
	StagingDir string
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
		shaSums, err := fetch(ctx, client, plan.Verification.ChecksumURL)
		if err != nil {
			return provider.BootPlan{}, fmt.Errorf("fetch SHA256SUMS: %w", err)
		}
		signature, err := fetch(ctx, client, plan.Verification.SignatureURL)
		if err != nil {
			return provider.BootPlan{}, fmt.Errorf("fetch SHA256SUMS.gpg: %w", err)
		}
		if err := verify.Artifact(verify.ArtifactInput{
			Artifact:  bytes.NewReader(shaSums),
			Signature: bytes.NewReader(signature),
			Keyring:   config.Keyring,
			Name:      "SHA256SUMS",
		}); err != nil {
			return provider.BootPlan{}, err
		}
		if err := requireChecksumEntry(shaSums, liveServerISO); err != nil {
			return provider.BootPlan{}, err
		}
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

func fetchStageVerify(ctx context.Context, client *http.Client, dir string, artifact provider.Artifact) (string, error) {
	data, err := fetch(ctx, client, artifact.URL)
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
