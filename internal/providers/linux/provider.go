// Package linux provides static Linux kernel/initrd catalog targets.
package linux

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/dotwaffle/bootup/internal/provider"
	"github.com/dotwaffle/bootup/internal/providerhttp"
	"github.com/dotwaffle/bootup/verify"
)

const providerID = "linux"

// Config configures the generic Linux provider.
type Config struct {
	Client  *http.Client
	Targets []provider.Target
}

// Provider exposes static Linux kernel/initrd targets from catalog metadata.
type Provider struct {
	client  *http.Client
	targets []provider.Target
}

// NewProvider creates a generic Linux provider.
func NewProvider(config Config) *Provider {
	return &Provider{
		client:  config.Client,
		targets: cloneTargets(config.Targets),
	}
}

// ID returns the provider ID.
func (*Provider) ID() string {
	return providerID
}

// Targets returns static Linux targets.
func (p *Provider) Targets(context.Context) ([]provider.Target, error) {
	return cloneTargets(p.targets), nil
}

// Plan returns a Linux kexec boot plan for target.
func (p *Provider) Plan(_ context.Context, target provider.Target) (provider.BootPlan, error) {
	selected, err := p.selectedTarget(target)
	if err != nil {
		return provider.BootPlan{}, err
	}
	if provider.ResolveBootAction(selected.Action) != provider.BootActionLinuxKexec {
		return provider.BootPlan{}, fmt.Errorf("unsupported Linux boot action %q for target %s", selected.Action, selected.ID)
	}
	baseURL := strings.TrimRight(selected.Source.BaseURL, "/")
	if baseURL == "" {
		return provider.BootPlan{}, fmt.Errorf("source base URL is required for Linux target %s", selected.ID)
	}
	if selected.Source.KernelPath == "" {
		return provider.BootPlan{}, fmt.Errorf("source kernel path is required for Linux target %s", selected.ID)
	}

	plan := provider.BootPlan{
		Target: selected,
		Action: provider.BootActionLinuxKexec,
		Kernel: provider.Artifact{
			Name: path.Base(selected.Source.KernelPath),
			URL:  joinURLPath(baseURL, selected.Source.KernelPath),
		},
		Cmdline: strings.ReplaceAll(selected.Source.Cmdline, "{base_url}", baseURL),
	}
	if selected.Source.InitrdPath != "" {
		plan.Initrd = provider.Artifact{
			Name: path.Base(selected.Source.InitrdPath),
			URL:  joinURLPath(baseURL, selected.Source.InitrdPath),
		}
	}
	return plan, nil
}

// Stage downloads, verifies, and stages artifacts for plan.
func (p *Provider) Stage(ctx context.Context, config provider.StageConfig) (provider.BootPlan, error) {
	return FetchAndStageArtifacts(ctx, FetchConfig{
		Plan:       config.Plan,
		Client:     p.client,
		StagingDir: config.StagingDir,
	})
}

// FetchConfig configures Linux artifact fetching and staging.
type FetchConfig struct {
	Plan       provider.BootPlan
	Client     *http.Client
	StagingDir string
}

// FetchAndStageArtifacts downloads Linux artifacts and stages them on disk.
func FetchAndStageArtifacts(ctx context.Context, config FetchConfig) (provider.BootPlan, error) {
	client := config.Client
	if client == nil {
		client = http.DefaultClient
	}
	if config.StagingDir == "" {
		return provider.BootPlan{}, errors.New("staging dir is required")
	}

	plan := config.Plan
	if plan.Kernel.URL == "" {
		return provider.BootPlan{}, errors.New("kernel URL is required")
	}
	if err := requireHTTPS(nonEmptyURLs(plan.Kernel.URL, plan.Initrd.URL)...); err != nil {
		return provider.BootPlan{}, err
	}

	var err error
	if plan.Kernel.Path, err = fetchStageVerify(ctx, client, config.StagingDir, plan.Kernel); err != nil {
		return provider.BootPlan{}, err
	}
	if plan.Initrd.URL != "" {
		if plan.Initrd.Path, err = fetchStageVerify(ctx, client, config.StagingDir, plan.Initrd); err != nil {
			return provider.BootPlan{}, err
		}
	}
	return plan, nil
}

func (p *Provider) selectedTarget(target provider.Target) (provider.Target, error) {
	if selected, ok := p.target(target.ID); ok {
		return selected, nil
	}
	if err := provider.ValidateTarget(providerID, target); err != nil {
		return provider.Target{}, err
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

func joinURLPath(baseURL string, elem string) string {
	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(elem, "/")
}

func requireHTTPS(rawURLs ...string) error {
	for _, rawURL := range rawURLs {
		parsed, err := url.Parse(rawURL)
		if err != nil {
			return fmt.Errorf("parse artifact URL: %w", err)
		}
		if parsed.Scheme != "https" {
			return fmt.Errorf("unverified Linux artifact URL must use https: %s", rawURL)
		}
	}
	return nil
}

func nonEmptyURLs(rawURLs ...string) []string {
	out := make([]string, 0, len(rawURLs))
	for _, rawURL := range rawURLs {
		if rawURL != "" {
			out = append(out, rawURL)
		}
	}
	return out
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
	name := artifact.Name
	if name == "" {
		name = path.Base(artifact.URL)
	}
	targetPath := filepath.Join(dir, name)
	if err := os.WriteFile(targetPath, data, 0o644); err != nil {
		return "", fmt.Errorf("stage %s: %w", name, err)
	}
	return targetPath, nil
}

func cloneTargets(targets []provider.Target) []provider.Target {
	return append([]provider.Target(nil), targets...)
}
