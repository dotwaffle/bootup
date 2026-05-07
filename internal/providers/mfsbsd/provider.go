// Package mfsbsd provides mfsBSD kboot targets.
package mfsbsd

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/dotwaffle/bootup/internal/provider"
	"github.com/dotwaffle/bootup/internal/providerhttp"
	"github.com/dotwaffle/bootup/verify"
	"github.com/u-root/u-root/pkg/mount"
	"github.com/u-root/u-root/pkg/mount/loop"
	"github.com/ulikunitz/xz"
)

const (
	providerID = "mfsbsd"
	targetID   = "mfsbsd-142-amd64"

	defaultBaseURL             = "https://mfsbsd.vx.sk/files/iso/14/amd64"
	defaultISOName             = "mfsbsd-14.2-RELEASE-amd64.iso"
	defaultISOSHA256           = "2803be01ef284cb4d58c9177475c7a20ac72292e4943bc91eb159c592bfc3b5c"
	defaultLoaderArchiveURL    = "https://download.freebsd.org/releases/amd64/amd64/15.0-RELEASE/base.txz"
	defaultLoaderArchiveSHA256 = "ac0c933cc02ee8af4da793f551e4a9a15cdcf0e67851290b1e8c19dd6d30bba8"
)

// Config configures the mfsBSD provider.
type Config struct {
	Client              *http.Client
	Targets             []provider.Target
	LoaderArchiveURL    string
	LoaderArchiveSHA256 string
	Extractor           ISOExtractor
}

// Provider exposes mfsBSD targets.
type Provider struct {
	client              *http.Client
	targets             []provider.Target
	loaderArchiveURL    string
	loaderArchiveSHA256 string
	extractor           ISOExtractor
}

// FetchConfig configures mfsBSD artifact fetching and staging.
type FetchConfig struct {
	Plan       provider.BootPlan
	Client     *http.Client
	StagingDir string
	Extractor  ISOExtractor
	Progress   provider.StageProgressFunc
}

// ISOExtractor extracts ISO filesystem contents to dest.
type ISOExtractor interface {
	Extract(context.Context, string, string) error
}

// LoopISOExtractor extracts an ISO by loop-mounting it read-only.
type LoopISOExtractor struct{}

// NewProvider creates an mfsBSD provider.
func NewProvider(config Config) *Provider {
	targets := cloneTargets(config.Targets)
	if config.Targets == nil {
		targets = defaultTargets()
	}
	loaderArchiveURL := strings.TrimSpace(config.LoaderArchiveURL)
	if loaderArchiveURL == "" {
		loaderArchiveURL = defaultLoaderArchiveURL
	}
	loaderArchiveSHA256 := strings.ToLower(strings.TrimSpace(config.LoaderArchiveSHA256))
	if loaderArchiveSHA256 == "" {
		loaderArchiveSHA256 = defaultLoaderArchiveSHA256
	}
	return &Provider{
		client:              config.Client,
		targets:             targets,
		loaderArchiveURL:    loaderArchiveURL,
		loaderArchiveSHA256: loaderArchiveSHA256,
		extractor:           config.Extractor,
	}
}

// ID returns the provider ID.
func (*Provider) ID() string {
	return providerID
}

// Targets returns static mfsBSD targets.
func (p *Provider) Targets(context.Context) ([]provider.Target, error) {
	return cloneTargets(p.targets), nil
}

// Plan returns a FreeBSD kboot plan for target.
func (p *Provider) Plan(_ context.Context, input provider.PlanInput) (provider.BootPlan, error) {
	selected, err := p.selectedTarget(input.Target)
	if err != nil {
		return provider.BootPlan{}, err
	}
	if provider.ResolveBootAction(selected.Action) != provider.BootActionFreeBSDKboot {
		return provider.BootPlan{}, fmt.Errorf("unsupported mfsBSD boot action %q for target %s", selected.Action, selected.ID)
	}
	if selected.Catalog.Distribution != providerID {
		return provider.BootPlan{}, fmt.Errorf("unsupported mfsBSD distribution %q for target %s", selected.Catalog.Distribution, selected.ID)
	}
	if selected.Catalog.Architecture != "amd64" {
		return provider.BootPlan{}, fmt.Errorf("unsupported mfsBSD architecture %q for target %s", selected.Catalog.Architecture, selected.ID)
	}

	baseURL := strings.TrimRight(selected.Source.BaseURL, "/")
	if baseURL == "" {
		return provider.BootPlan{}, fmt.Errorf("source base URL is required for mfsBSD target %s", selected.ID)
	}
	isoName := selected.Source.ISOName
	if isoName == "" {
		return provider.BootPlan{}, fmt.Errorf("source ISO name is required for mfsBSD target %s", selected.ID)
	}
	if selected.Source.ISOSHA256 == "" {
		return provider.BootPlan{}, fmt.Errorf("source ISO SHA256 is required for mfsBSD target %s", selected.ID)
	}

	plan := provider.BootPlan{
		Target: selected,
		Action: provider.BootActionFreeBSDKboot,
		FreeBSDKboot: provider.FreeBSDKbootPlan{
			Payload: provider.Artifact{
				Name:   isoName,
				URL:    joinURLPath(baseURL, isoName),
				SHA256: selected.Source.ISOSHA256,
			},
			LoaderArchive: provider.Artifact{
				Name:   artifactNameFromURL(p.loaderArchiveURL, "base.txz"),
				URL:    p.loaderArchiveURL,
				SHA256: p.loaderArchiveSHA256,
			},
		},
	}
	return provider.ApplySelectedOptions(plan, input.Options)
}

// Stage downloads, verifies, and stages artifacts for plan.
func (p *Provider) Stage(ctx context.Context, config provider.StageConfig) (provider.BootPlan, error) {
	extractor := p.extractor
	if extractor == nil {
		extractor = FileISOExtractor{}
	}
	return FetchAndStageArtifacts(ctx, FetchConfig{
		Plan:       config.Plan,
		Client:     p.client,
		StagingDir: config.StagingDir,
		Extractor:  extractor,
		Progress:   config.Progress,
	})
}

// FetchAndStageArtifacts downloads mfsBSD and loader artifacts, verifies
// pinned hashes, and stages a loader.kboot hostfs payload tree.
func FetchAndStageArtifacts(ctx context.Context, config FetchConfig) (provider.BootPlan, error) {
	client := config.Client
	if client == nil {
		client = http.DefaultClient
	}
	if config.StagingDir == "" {
		return provider.BootPlan{}, errors.New("staging dir is required")
	}
	if provider.ResolveBootAction(config.Plan.Action) != provider.BootActionFreeBSDKboot {
		return provider.BootPlan{}, fmt.Errorf("unsupported mfsBSD boot action %q", config.Plan.Action)
	}
	extractor := config.Extractor
	if extractor == nil {
		extractor = FileISOExtractor{}
	}

	plan := config.Plan
	kboot := plan.FreeBSDKboot
	selectedArgs := append([]string(nil), kboot.Args...)
	if err := validateArtifact("mfsBSD ISO", kboot.Payload); err != nil {
		return provider.BootPlan{}, err
	}
	if err := validateArtifact("FreeBSD loader archive", kboot.LoaderArchive); err != nil {
		return provider.BootPlan{}, err
	}
	if err := requireHTTPS(kboot.Payload.URL, kboot.LoaderArchive.URL); err != nil {
		return provider.BootPlan{}, err
	}
	if err := os.MkdirAll(config.StagingDir, 0o755); err != nil {
		return provider.BootPlan{}, fmt.Errorf("create staging dir: %w", err)
	}

	var err error
	if plan.FreeBSDKboot.Payload.Path, err = fetchStageVerify(ctx, client, config.StagingDir, kboot.Payload, config.Progress); err != nil {
		return provider.BootPlan{}, err
	}
	if plan.FreeBSDKboot.LoaderArchive.Path, err = fetchStageVerify(ctx, client, config.StagingDir, kboot.LoaderArchive, config.Progress); err != nil {
		return provider.BootPlan{}, err
	}

	payloadRoot, err := os.MkdirTemp(config.StagingDir, "mfsbsd-root-")
	if err != nil {
		return provider.BootPlan{}, fmt.Errorf("create mfsBSD payload root: %w", err)
	}
	if err := reportProgress(config.Progress, provider.StageOperationExtract, provider.StageStateStarted, "mfsBSD ISO"); err != nil {
		return provider.BootPlan{}, err
	}
	if err := extractor.Extract(ctx, plan.FreeBSDKboot.Payload.Path, payloadRoot); err != nil {
		return provider.BootPlan{}, fmt.Errorf("extract mfsBSD ISO: %w", err)
	}
	if err := reportProgress(config.Progress, provider.StageOperationExtract, provider.StageStateCompleted, "mfsBSD ISO"); err != nil {
		return provider.BootPlan{}, err
	}
	if err := normalizeMemoryRoot(payloadRoot); err != nil {
		return provider.BootPlan{}, err
	}

	loaderDir, err := os.MkdirTemp(config.StagingDir, "freebsd-kboot-")
	if err != nil {
		return provider.BootPlan{}, fmt.Errorf("create FreeBSD kboot staging dir: %w", err)
	}
	archiveData, err := os.ReadFile(plan.FreeBSDKboot.LoaderArchive.Path)
	if err != nil {
		return provider.BootPlan{}, fmt.Errorf("read FreeBSD loader archive: %w", err)
	}
	if err := reportProgress(config.Progress, provider.StageOperationExtract, provider.StageStateStarted, "FreeBSD loader archive"); err != nil {
		return provider.BootPlan{}, err
	}
	loader, loaderHelp, err := extractLoaderArchive(archiveData, loaderDir)
	if err != nil {
		return provider.BootPlan{}, err
	}
	if err := reportProgress(config.Progress, provider.StageOperationExtract, provider.StageStateCompleted, "FreeBSD loader archive"); err != nil {
		return provider.BootPlan{}, err
	}

	plan.FreeBSDKboot.Loader = provider.Artifact{Name: "loader.kboot", Path: loader}
	plan.FreeBSDKboot.LoaderHelp = provider.Artifact{Name: "loader.help.kboot", Path: loaderHelp}
	plan.FreeBSDKboot.PayloadRoot = payloadRoot
	plan.FreeBSDKboot.Args = append(defaultLoaderArgs(payloadRoot), selectedArgs...)
	return plan, nil
}

// Extract loop-mounts isoPath read-only and copies its contents to dest.
func (LoopISOExtractor) Extract(ctx context.Context, isoPath string, dest string) (err error) {
	if strings.TrimSpace(isoPath) == "" {
		return errors.New("ISO path is required")
	}
	if strings.TrimSpace(dest) == "" {
		return errors.New("extract destination is required")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	mountDir, err := os.MkdirTemp(filepath.Dir(dest), ".mfsbsd-iso-")
	if err != nil {
		return fmt.Errorf("create ISO mount dir: %w", err)
	}
	defer func() {
		if removeErr := os.RemoveAll(mountDir); err == nil && removeErr != nil {
			err = fmt.Errorf("remove ISO mount dir: %w", removeErr)
		}
	}()

	loopDevice, err := loop.New(isoPath, "iso9660", "")
	if err != nil {
		return fmt.Errorf("create loop device: %w", err)
	}
	defer func() {
		if freeErr := loopDevice.Free(); err == nil && freeErr != nil {
			err = fmt.Errorf("free loop device: %w", freeErr)
		}
	}()

	mountPoint, err := loopDevice.Mount(mountDir, mount.MS_RDONLY)
	if err != nil {
		return fmt.Errorf("mount ISO: %w", err)
	}
	defer func() {
		if unmountErr := mountPoint.Unmount(0); err == nil && unmountErr != nil {
			err = fmt.Errorf("unmount ISO: %w", unmountErr)
		}
	}()

	if err := copyTree(ctx, mountDir, dest); err != nil {
		return fmt.Errorf("copy ISO contents: %w", err)
	}
	return nil
}

func defaultTargets() []provider.Target {
	return []provider.Target{{
		ID:         targetID,
		ProviderID: providerID,
		Name:       "mfsBSD 14.2 amd64",
		Action:     provider.BootActionFreeBSDKboot,
		Catalog: provider.CatalogEntry{
			Distribution: providerID,
			Release:      "14.2",
			Architecture: "amd64",
			Kind:         "rescue",
		},
		Source: provider.SourceEntry{
			BaseURL:   defaultBaseURL,
			ISOName:   defaultISOName,
			ISOSHA256: defaultISOSHA256,
		},
		Lifecycle: provider.LifecycleEntry{
			Status: provider.LifecycleSupported,
			Source: "catalog",
		},
		Options: []provider.TargetOption{
			{
				ID:       "hostname",
				Label:    "Hostname",
				Type:     provider.TargetOptionString,
				Template: "mfsbsd.hostname={value}",
			},
		},
	}}
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

func cloneTargets(targets []provider.Target) []provider.Target {
	return append([]provider.Target(nil), targets...)
}

func joinURLPath(baseURL string, elem string) string {
	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(elem, "/")
}

func artifactNameFromURL(rawURL string, fallback string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fallback
	}
	name := path.Base(parsed.Path)
	if name == "." || name == "/" {
		return fallback
	}
	return name
}

func validateArtifact(label string, artifact provider.Artifact) error {
	if strings.TrimSpace(artifact.URL) == "" {
		return fmt.Errorf("%s URL is required", label)
	}
	if strings.TrimSpace(artifact.SHA256) == "" {
		return fmt.Errorf("%s SHA256 is required", label)
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
			return fmt.Errorf("unverified mfsBSD artifact URL must use https: %s", rawURL)
		}
	}
	return nil
}

func fetchStageVerify(ctx context.Context, client *http.Client, dir string, artifact provider.Artifact, progress provider.StageProgressFunc) (string, error) {
	artifactLabel := artifact.Name
	if artifactLabel == "" {
		artifactLabel = artifactNameFromURL(artifact.URL, "")
	}
	if err := reportProgress(progress, provider.StageOperationFetch, provider.StageStateStarted, artifactLabel); err != nil {
		return "", err
	}
	data, err := providerhttp.Fetch(ctx, client, artifact.URL)
	if err != nil {
		return "", fmt.Errorf("fetch %s: %w", artifact.Name, err)
	}
	if err := reportProgress(progress, provider.StageOperationFetch, provider.StageStateCompleted, artifactLabel); err != nil {
		return "", err
	}
	name := artifact.Name
	if name == "" {
		name = artifactNameFromURL(artifact.URL, "")
	}
	if name == "" || filepath.Base(name) != name {
		return "", fmt.Errorf("artifact name %q must be a filename", name)
	}
	if err := reportProgress(progress, provider.StageOperationVerify, provider.StageStateStarted, name); err != nil {
		return "", err
	}
	if err := verify.SHA256(verify.HashInput{
		Artifact:       bytes.NewReader(data),
		ExpectedSHA256: artifact.SHA256,
		Name:           name,
	}); err != nil {
		return "", err
	}
	if err := reportProgress(progress, provider.StageOperationVerify, provider.StageStateCompleted, name); err != nil {
		return "", err
	}
	if err := reportProgress(progress, provider.StageOperationWrite, provider.StageStateStarted, name); err != nil {
		return "", err
	}
	targetPath := filepath.Join(dir, name)
	if err := os.WriteFile(targetPath, data, 0o644); err != nil {
		return "", fmt.Errorf("stage %s: %w", name, err)
	}
	if err := reportProgress(progress, provider.StageOperationWrite, provider.StageStateCompleted, name); err != nil {
		return "", err
	}
	return targetPath, nil
}

func reportProgress(progress provider.StageProgressFunc, operation provider.StageOperation, state provider.StageState, artifact string) error {
	return provider.ReportStageProgress(progress, provider.StageProgress{
		Operation: operation,
		State:     state,
		Artifact:  artifact,
	})
}

func normalizeMemoryRoot(root string) error {
	for _, name := range []string{
		filepath.Join("boot", "kernel", "kernel"),
		"mfsroot",
	} {
		if err := normalizeGzipFile(filepath.Join(root, name)); err != nil {
			return err
		}
	}
	return nil
}

func normalizeGzipFile(targetPath string) error {
	if _, err := os.Stat(targetPath); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat %s: %w", targetPath, err)
	}
	sourcePath := targetPath + ".gz"
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open %s: %w", sourcePath, err)
	}
	defer func() { _ = source.Close() }()
	gzipReader, err := gzip.NewReader(source)
	if err != nil {
		return fmt.Errorf("open gzip %s: %w", sourcePath, err)
	}
	defer func() { _ = gzipReader.Close() }()

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("create %s parent: %w", targetPath, err)
	}
	temp, err := os.CreateTemp(filepath.Dir(targetPath), "."+filepath.Base(targetPath)+".")
	if err != nil {
		return fmt.Errorf("create temp %s: %w", targetPath, err)
	}
	tempPath := temp.Name()
	if _, err := io.Copy(temp, gzipReader); err != nil {
		_ = temp.Close()
		_ = os.Remove(tempPath)
		return fmt.Errorf("decompress %s: %w", sourcePath, err)
	}
	if err := temp.Close(); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("close %s: %w", tempPath, err)
	}
	if err := os.Chmod(tempPath, 0o644); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("chmod %s: %w", tempPath, err)
	}
	if err := os.Rename(tempPath, targetPath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("stage %s: %w", targetPath, err)
	}
	return nil
}

func extractLoaderArchive(data []byte, dir string) (string, string, error) {
	xzReader, err := xz.NewReader(bytes.NewReader(data))
	if err != nil {
		return "", "", fmt.Errorf("open FreeBSD loader archive xz: %w", err)
	}
	tarReader := tar.NewReader(xzReader)
	loaderPath := filepath.Join(dir, "loader.kboot")
	loaderHelpPath := filepath.Join(dir, "loader.help.kboot")
	var foundLoader, foundHelp bool
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", "", fmt.Errorf("read FreeBSD loader archive: %w", err)
		}
		if !header.FileInfo().Mode().IsRegular() {
			continue
		}
		name := strings.TrimPrefix(path.Clean(header.Name), "./")
		switch name {
		case "boot/loader.kboot":
			if err := writeArchiveMember(loaderPath, tarReader, 0o555); err != nil {
				return "", "", err
			}
			foundLoader = true
		case "boot/loader.help.kboot":
			if err := writeArchiveMember(loaderHelpPath, tarReader, 0o444); err != nil {
				return "", "", err
			}
			foundHelp = true
		}
	}
	if !foundLoader {
		return "", "", errors.New("FreeBSD loader archive missing boot/loader.kboot")
	}
	if !foundHelp {
		return "", "", errors.New("FreeBSD loader archive missing boot/loader.help.kboot")
	}
	return loaderPath, loaderHelpPath, nil
}

func writeArchiveMember(targetPath string, reader io.Reader, mode fs.FileMode) error {
	temp, err := os.CreateTemp(filepath.Dir(targetPath), "."+filepath.Base(targetPath)+".")
	if err != nil {
		return fmt.Errorf("create temp %s: %w", targetPath, err)
	}
	tempPath := temp.Name()
	if _, err := io.Copy(temp, reader); err != nil {
		_ = temp.Close()
		_ = os.Remove(tempPath)
		return fmt.Errorf("extract %s: %w", targetPath, err)
	}
	if err := temp.Close(); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("close %s: %w", tempPath, err)
	}
	if err := os.Chmod(tempPath, mode); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("chmod %s: %w", tempPath, err)
	}
	if err := os.Rename(tempPath, targetPath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("stage %s: %w", targetPath, err)
	}
	return nil
}

func defaultLoaderArgs(payloadRoot string) []string {
	return []string{
		"hostfs_root=" + payloadRoot,
		"bootdev=host:/",
		"boot_serial=YES",
		"boot_multicons=YES",
		"boot_verbose=YES",
		"autoboot_delay=0",
		"beastie_disable=YES",
		"mfsbsd.autodhcp=YES",
		"mfsbsd.hostname=mfsbsd",
	}
}

func copyTree(ctx context.Context, source string, dest string) error {
	return filepath.WalkDir(source, func(sourcePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		rel, err := filepath.Rel(source, sourcePath)
		if err != nil {
			return fmt.Errorf("rel %s: %w", sourcePath, err)
		}
		if rel == "." {
			return os.MkdirAll(dest, 0o755)
		}
		targetPath := filepath.Join(dest, rel)
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("stat %s: %w", sourcePath, err)
		}
		switch {
		case entry.IsDir():
			return os.MkdirAll(targetPath, 0o755)
		case info.Mode().Type() == 0:
			return copyFile(sourcePath, targetPath, info.Mode().Perm())
		case info.Mode()&os.ModeSymlink != 0:
			return copySymlink(sourcePath, targetPath)
		default:
			return nil
		}
	})
}

func copyFile(sourcePath string, targetPath string, mode fs.FileMode) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open %s: %w", sourcePath, err)
	}
	defer func() { _ = source.Close() }()
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("create %s parent: %w", targetPath, err)
	}
	target, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("create %s: %w", targetPath, err)
	}
	if _, err := io.Copy(target, source); err != nil {
		_ = target.Close()
		return fmt.Errorf("copy %s: %w", sourcePath, err)
	}
	if err := target.Close(); err != nil {
		return fmt.Errorf("close %s: %w", targetPath, err)
	}
	return nil
}

func copySymlink(sourcePath string, targetPath string) error {
	linkTarget, err := os.Readlink(sourcePath)
	if err != nil {
		return fmt.Errorf("readlink %s: %w", sourcePath, err)
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("create %s parent: %w", targetPath, err)
	}
	if err := os.Symlink(linkTarget, targetPath); err != nil {
		return fmt.Errorf("symlink %s: %w", targetPath, err)
	}
	return nil
}
