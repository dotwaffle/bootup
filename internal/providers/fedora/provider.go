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
	"strings"

	"github.com/dotwaffle/bootup/internal/provider"
	"github.com/dotwaffle/bootup/internal/providerhttp"
	"github.com/dotwaffle/bootup/verify"
)

const (
	defaultReleaseURL = "https://download.fedoraproject.org/pub/fedora/linux/releases/44/Server/x86_64/os"

	targetID        = "fedora-44-amd64-server-netboot"
	providerID      = "fedora"
	kernelStageName = "vmlinuz"
	initrdStageName = "initrd.img"
)

// Config configures the Fedora provider.
type Config struct {
	ReleaseURL   string
	Client       *http.Client
	KernelSHA256 string
	InitrdSHA256 string
	Targets      []provider.Target
}

// Provider exposes Fedora Server netboot targets.
type Provider struct {
	releaseURL           string
	client               *http.Client
	kernelSHA256         string
	initrdSHA256         string
	targets              []provider.Target
	releaseURLConfigured bool
}

// NewProvider creates a Fedora provider.
func NewProvider(config Config) *Provider {
	releaseURL := strings.TrimRight(config.ReleaseURL, "/")
	if releaseURL == "" {
		releaseURL = defaultReleaseURL
	}
	targets := cloneTargets(config.Targets)
	if config.Targets == nil {
		targets = defaultTargets()
	}
	return &Provider{
		releaseURL:           releaseURL,
		client:               config.Client,
		kernelSHA256:         config.KernelSHA256,
		initrdSHA256:         config.InitrdSHA256,
		targets:              targets,
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

// Plan returns a boot plan for target.
func (p *Provider) Plan(_ context.Context, target provider.Target) (provider.BootPlan, error) {
	selected, err := p.selectedTarget(target)
	if err != nil {
		return provider.BootPlan{}, err
	}
	architecture := selected.Catalog.Architecture
	if architecture != "amd64" {
		return provider.BootPlan{}, fmt.Errorf("unsupported Fedora architecture %q for target %s", architecture, selected.ID)
	}
	releaseURL := p.targetBaseURL(selected)

	return provider.BootPlan{
		Target: selected,
		Kernel: provider.Artifact{
			Name:   kernelStageName,
			URL:    releaseURL + "/images/pxeboot/vmlinuz",
			SHA256: p.kernelSHA256,
		},
		Initrd: provider.Artifact{
			Name:   initrdStageName,
			URL:    releaseURL + "/images/pxeboot/initrd.img",
			SHA256: p.initrdSHA256,
		},
		Cmdline: "inst.repo=" + releaseURL + " ip=dhcp console=ttyS0",
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
