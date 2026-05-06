package ubuntu_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/dotwaffle/bootup/internal/provider"
	"github.com/dotwaffle/bootup/internal/providers/ubuntu"
)

func TestProviderTargetsUbuntu2604AMD64(t *testing.T) {
	t.Parallel()

	targets, err := ubuntu.NewProvider(ubuntu.Config{}).Targets(context.Background())
	if err != nil {
		t.Fatalf("targets: %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("targets length = %d, want 1", len(targets))
	}
	target := targets[0]
	if target.ID != "ubuntu-2604-amd64-netboot" {
		t.Fatalf("target ID = %q, want Ubuntu 26.04", target.ID)
	}
	if target.ProviderID != "ubuntu" {
		t.Fatalf("provider ID = %q, want ubuntu", target.ProviderID)
	}
	if target.Catalog.Architecture != "amd64" {
		t.Fatalf("architecture = %q, want amd64", target.Catalog.Architecture)
	}
	if target.Catalog.Distribution != "ubuntu" {
		t.Fatalf("distribution = %q, want ubuntu", target.Catalog.Distribution)
	}
	if target.Catalog.Release != "26.04" {
		t.Fatalf("release = %q, want 26.04", target.Catalog.Release)
	}
	if target.Catalog.Kind != "installer" {
		t.Fatalf("kind = %q, want installer", target.Catalog.Kind)
	}
}

func TestProviderDiscoveryFamily(t *testing.T) {
	t.Parallel()

	family := ubuntu.NewProvider(ubuntu.Config{}).DiscoveryFamily()

	if family.ID != "ubuntu" {
		t.Fatalf("family ID = %q, want ubuntu", family.ID)
	}
	if family.ProviderID != "ubuntu" {
		t.Fatalf("family provider ID = %q, want ubuntu", family.ProviderID)
	}
	if family.Name != "Ubuntu" {
		t.Fatalf("family name = %q, want Ubuntu", family.Name)
	}
}

func TestProviderDiscoversAMD64NetbootTargets(t *testing.T) {
	t.Parallel()

	shaSums := []byte(strings.Repeat("a", 64) + " *ubuntu-24.04.4-live-server-amd64.iso\n")
	p := ubuntu.NewProvider(ubuntu.Config{
		DiscoveryURL: "https://releases.example/releases/",
		Client: &http.Client{Transport: responseMap{
			"https://releases.example/releases/":                           []byte(`<a href="../">Parent</a><a href="24.04/">Ubuntu 24.04</a><a href="22.04/">Ubuntu 22.04</a>`),
			"https://releases.example/releases/24.04/SHA256SUMS":           shaSums,
			"https://releases.example/releases/24.04/netboot/amd64/linux":  []byte("kernel"),
			"https://releases.example/releases/24.04/netboot/amd64/initrd": []byte("initrd"),
			"https://releases.example/releases/22.04/SHA256SUMS":           []byte(strings.Repeat("b", 64) + " *ubuntu-22.04.5-live-server-amd64.iso\n"),
		}},
		Lifecycle: map[string]provider.LifecycleEntry{
			"24.04.4": {
				Status: provider.LifecycleSupported,
				Source: "operator",
				Date:   "2029-05-31",
			},
		},
	})

	targets, err := p.DiscoverTargets(context.Background())
	if err != nil {
		t.Fatalf("discover targets: %v", err)
	}

	if len(targets) != 1 {
		t.Fatalf("targets length = %d, want 1: %#v", len(targets), targets)
	}
	target := targets[0]
	if target.ID != "ubuntu-24044-amd64-netboot" {
		t.Fatalf("target ID = %q, want Ubuntu 24.04.4", target.ID)
	}
	if target.Source.BaseURL != "https://releases.example/releases/24.04" {
		t.Fatalf("source base URL = %q", target.Source.BaseURL)
	}
	if target.Source.ISOName != "ubuntu-24.04.4-live-server-amd64.iso" {
		t.Fatalf("source ISO name = %q", target.Source.ISOName)
	}
	if target.Lifecycle.Status != provider.LifecycleSupported || target.Lifecycle.Source != "operator" || target.Lifecycle.Date != "2029-05-31" {
		t.Fatalf("lifecycle = %#v, want configured supported entry", target.Lifecycle)
	}
}

func TestProviderDiscoveryUsesTimeout(t *testing.T) {
	t.Parallel()

	p := ubuntu.NewProvider(ubuntu.Config{
		DiscoveryURL:     "https://releases.example/releases/",
		Client:           &http.Client{Transport: blockingTransport{}},
		DiscoveryTimeout: time.Nanosecond,
	})

	_, err := p.DiscoverTargets(context.Background())
	if err == nil {
		t.Fatal("discover targets succeeded, want timeout error")
	}
}

func TestProviderPlanResolvesReleaseURLs(t *testing.T) {
	t.Parallel()

	p := ubuntu.NewProvider(ubuntu.Config{
		ReleaseURL:   "https://mirror.example/26.04",
		KernelSHA256: strings.Repeat("a", 64),
		InitrdSHA256: strings.Repeat("b", 64),
	})
	plan, err := p.Plan(context.Background(), provider.PlanInput{
		Target: provider.Target{
			ID:         "ubuntu-2604-amd64-netboot",
			ProviderID: "ubuntu",
		},
	})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}

	if plan.Kernel.URL != "https://mirror.example/26.04/netboot/amd64/linux" {
		t.Fatalf("kernel URL = %q", plan.Kernel.URL)
	}
	if plan.Initrd.URL != "https://mirror.example/26.04/netboot/amd64/initrd" {
		t.Fatalf("initrd URL = %q", plan.Initrd.URL)
	}
	if plan.Verification.ChecksumURL != "https://mirror.example/26.04/SHA256SUMS" {
		t.Fatalf("checksum URL = %q", plan.Verification.ChecksumURL)
	}
	if plan.Verification.SignatureURL != "https://mirror.example/26.04/SHA256SUMS.gpg" {
		t.Fatalf("signature URL = %q", plan.Verification.SignatureURL)
	}
	if !strings.Contains(plan.Cmdline, "url=https://mirror.example/26.04/ubuntu-26.04-live-server-amd64.iso") {
		t.Fatalf("cmdline = %q, want ISO URL", plan.Cmdline)
	}
	if plan.Kernel.SHA256 != strings.Repeat("a", 64) {
		t.Fatalf("kernel sha256 = %q", plan.Kernel.SHA256)
	}
}

func TestProviderPlanAcceptsDiscoveredTarget(t *testing.T) {
	t.Parallel()

	target := ubuntuTarget("24.04.4")
	target.Source = provider.SourceEntry{
		BaseURL: "https://releases.example/24.04",
		ISOName: "ubuntu-24.04.4-live-server-amd64.iso",
	}
	p := ubuntu.NewProvider(ubuntu.Config{
		ReleaseURL: "https://fallback.example/26.04",
	})

	plan, err := p.Plan(context.Background(), provider.PlanInput{Target: target})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}

	if plan.Target.ID != "ubuntu-24044-amd64-netboot" {
		t.Fatalf("planned target = %q, want discovered target", plan.Target.ID)
	}
	if plan.Kernel.URL != "https://releases.example/24.04/netboot/amd64/linux" {
		t.Fatalf("kernel URL = %q", plan.Kernel.URL)
	}
}

func TestProviderPlanResolvesTargetSourceURLs(t *testing.T) {
	t.Parallel()

	target := ubuntuTarget("24.04.4")
	target.Source = provider.SourceEntry{
		BaseURL: "https://releases.example/24.04/",
		ISOName: "ubuntu-24.04.4-live-server-amd64.iso",
	}
	p := ubuntu.NewProvider(ubuntu.Config{
		ReleaseURL: "https://mirror.example/26.04",
		Targets:    []provider.Target{target},
	})

	plan, err := p.Plan(context.Background(), provider.PlanInput{Target: target})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}

	if plan.Kernel.URL != "https://releases.example/24.04/netboot/amd64/linux" {
		t.Fatalf("kernel URL = %q", plan.Kernel.URL)
	}
	if plan.Initrd.URL != "https://releases.example/24.04/netboot/amd64/initrd" {
		t.Fatalf("initrd URL = %q", plan.Initrd.URL)
	}
	if plan.Verification.ChecksumURL != "https://releases.example/24.04/SHA256SUMS" {
		t.Fatalf("checksum URL = %q", plan.Verification.ChecksumURL)
	}
	if !strings.Contains(plan.Cmdline, "url=https://releases.example/24.04/ubuntu-24.04.4-live-server-amd64.iso") {
		t.Fatalf("cmdline = %q, want source ISO URL", plan.Cmdline)
	}
}

func TestFetchAndStageArtifactsAllowsHTTPSOnlyNetboot(t *testing.T) {
	t.Parallel()

	kernel := []byte("kernel")
	initrd := []byte("initrd")
	plan := provider.BootPlan{
		Target: provider.Target{ID: "ubuntu-2604-amd64-netboot", ProviderID: "ubuntu"},
		Kernel: provider.Artifact{Name: "linux", URL: "https://mirror.example/26.04/netboot/amd64/linux"},
		Initrd: provider.Artifact{Name: "initrd", URL: "https://mirror.example/26.04/netboot/amd64/initrd"},
	}
	client := &http.Client{Transport: responseMap{
		"https://mirror.example/26.04/netboot/amd64/linux":  kernel,
		"https://mirror.example/26.04/netboot/amd64/initrd": initrd,
	}}

	staged, err := ubuntu.FetchAndStageArtifacts(context.Background(), ubuntu.FetchConfig{
		Plan:       plan,
		Client:     client,
		StagingDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("fetch and stage: %v", err)
	}
	gotKernel, err := os.ReadFile(staged.Kernel.Path)
	if err != nil {
		t.Fatalf("read staged kernel: %v", err)
	}
	if string(gotKernel) != string(kernel) {
		t.Fatalf("staged kernel = %q, want %q", gotKernel, kernel)
	}
}

func TestFetchAndStageArtifactsRejectsUnverifiedHTTP(t *testing.T) {
	t.Parallel()

	plan := provider.BootPlan{
		Target: provider.Target{ID: "ubuntu-2604-amd64-netboot", ProviderID: "ubuntu"},
		Kernel: provider.Artifact{Name: "linux", URL: "http://mirror.example/26.04/netboot/amd64/linux"},
		Initrd: provider.Artifact{Name: "initrd", URL: "http://mirror.example/26.04/netboot/amd64/initrd"},
	}

	_, err := ubuntu.FetchAndStageArtifacts(context.Background(), ubuntu.FetchConfig{
		Plan:       plan,
		StagingDir: t.TempDir(),
	})
	if err == nil {
		t.Fatal("fetch and stage succeeded, want https failure")
	}
}

func TestFetchAndStageArtifactsVerifiesSignedMetadataAndArtifacts(t *testing.T) {
	t.Parallel()

	kernel := []byte("kernel")
	initrd := []byte("initrd")
	iso := []byte("iso")
	kernelSum := sha256.Sum256(kernel)
	initrdSum := sha256.Sum256(initrd)
	isoSum := sha256.Sum256(iso)
	shaSums := fmt.Appendf(nil, "%x *ubuntu-26.04-live-server-amd64.iso\n", isoSum)
	keyring, signature := signedSHA256Sums(t, shaSums)

	client := &http.Client{Transport: responseMap{
		"https://mirror.example/26.04/SHA256SUMS":           shaSums,
		"https://mirror.example/26.04/SHA256SUMS.gpg":       signature,
		"https://mirror.example/26.04/netboot/amd64/linux":  kernel,
		"https://mirror.example/26.04/netboot/amd64/initrd": initrd,
	}}

	p := ubuntu.NewProvider(ubuntu.Config{
		ReleaseURL:   "https://mirror.example/26.04",
		KernelSHA256: hex.EncodeToString(kernelSum[:]),
		InitrdSHA256: hex.EncodeToString(initrdSum[:]),
	})
	plan, err := p.Plan(context.Background(), provider.PlanInput{
		Target: provider.Target{
			ID:         "ubuntu-2604-amd64-netboot",
			ProviderID: "ubuntu",
		},
	})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}

	staged, err := ubuntu.FetchAndStageArtifacts(context.Background(), ubuntu.FetchConfig{
		Plan:       plan,
		Client:     client,
		Keyring:    bytes.NewReader(keyring),
		StagingDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("fetch and stage: %v", err)
	}

	gotKernel, err := os.ReadFile(staged.Kernel.Path)
	if err != nil {
		t.Fatalf("read staged kernel: %v", err)
	}
	if string(gotKernel) != string(kernel) {
		t.Fatalf("staged kernel = %q, want %q", gotKernel, kernel)
	}
	if filepath.Base(staged.Initrd.Path) != "initrd" {
		t.Fatalf("staged initrd path = %q, want initrd", staged.Initrd.Path)
	}
}

func TestFetchAndStageArtifactsUsesTargetSourceISOName(t *testing.T) {
	t.Parallel()

	kernel := []byte("kernel")
	initrd := []byte("initrd")
	isoSum := sha256.Sum256([]byte("iso"))
	kernelSum := sha256.Sum256(kernel)
	initrdSum := sha256.Sum256(initrd)
	shaSums := fmt.Appendf(nil, "%x *ubuntu-24.04.4-live-server-amd64.iso\n", isoSum)
	keyring, signature := signedSHA256Sums(t, shaSums)

	client := &http.Client{Transport: responseMap{
		"https://releases.example/24.04/SHA256SUMS":           shaSums,
		"https://releases.example/24.04/SHA256SUMS.gpg":       signature,
		"https://releases.example/24.04/netboot/amd64/linux":  kernel,
		"https://releases.example/24.04/netboot/amd64/initrd": initrd,
	}}
	target := ubuntuTarget("24.04.4")
	target.Source = provider.SourceEntry{
		BaseURL: "https://releases.example/24.04",
		ISOName: "ubuntu-24.04.4-live-server-amd64.iso",
	}
	p := ubuntu.NewProvider(ubuntu.Config{
		Targets:      []provider.Target{target},
		KernelSHA256: hex.EncodeToString(kernelSum[:]),
		InitrdSHA256: hex.EncodeToString(initrdSum[:]),
	})
	plan, err := p.Plan(context.Background(), provider.PlanInput{Target: target})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}

	staged, err := ubuntu.FetchAndStageArtifacts(context.Background(), ubuntu.FetchConfig{
		Plan:       plan,
		Client:     client,
		Keyring:    bytes.NewReader(keyring),
		StagingDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("fetch and stage: %v", err)
	}
	if filepath.Base(staged.Kernel.Path) != "linux" {
		t.Fatalf("staged kernel path = %q, want linux", staged.Kernel.Path)
	}
}

func TestProviderStageUsesConfiguredTrust(t *testing.T) {
	t.Parallel()

	kernel := []byte("kernel")
	initrd := []byte("initrd")
	isoSum := sha256.Sum256([]byte("iso"))
	kernelSum := sha256.Sum256(kernel)
	initrdSum := sha256.Sum256(initrd)
	shaSums := fmt.Appendf(nil, "%x *ubuntu-26.04-live-server-amd64.iso\n", isoSum)
	keyring, signature := signedSHA256Sums(t, shaSums)

	client := &http.Client{Transport: responseMap{
		"https://mirror.example/26.04/SHA256SUMS":           shaSums,
		"https://mirror.example/26.04/SHA256SUMS.gpg":       signature,
		"https://mirror.example/26.04/netboot/amd64/linux":  kernel,
		"https://mirror.example/26.04/netboot/amd64/initrd": initrd,
	}}
	p := ubuntu.NewProvider(ubuntu.Config{
		ReleaseURL:   "https://mirror.example/26.04",
		Client:       client,
		Keyring:      keyring,
		KernelSHA256: hex.EncodeToString(kernelSum[:]),
		InitrdSHA256: hex.EncodeToString(initrdSum[:]),
	})
	plan, err := p.Plan(context.Background(), provider.PlanInput{
		Target: provider.Target{
			ID:         "ubuntu-2604-amd64-netboot",
			ProviderID: "ubuntu",
		},
	})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}

	staged, err := p.Stage(context.Background(), provider.StageConfig{
		Plan:       plan,
		StagingDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("stage: %v", err)
	}
	if filepath.Base(staged.Kernel.Path) != "linux" {
		t.Fatalf("staged kernel path = %q, want linux", staged.Kernel.Path)
	}
}

func signedSHA256Sums(t *testing.T, sums []byte) ([]byte, []byte) {
	t.Helper()

	entity, err := openpgp.NewEntity("Ubuntu Release", "", "ubuntu@example.test", nil)
	if err != nil {
		t.Fatalf("new entity: %v", err)
	}

	var signature bytes.Buffer
	if err := openpgp.DetachSign(&signature, entity, bytes.NewReader(sums), nil); err != nil {
		t.Fatalf("detach sign: %v", err)
	}

	var keyring bytes.Buffer
	armorWriter, err := armor.Encode(&keyring, openpgp.PublicKeyType, nil)
	if err != nil {
		t.Fatalf("armor keyring: %v", err)
	}
	if err := entity.Serialize(armorWriter); err != nil {
		t.Fatalf("serialize keyring: %v", err)
	}
	if err := armorWriter.Close(); err != nil {
		t.Fatalf("close armor: %v", err)
	}

	return keyring.Bytes(), signature.Bytes()
}

type responseMap map[string][]byte

func (m responseMap) RoundTrip(request *http.Request) (*http.Response, error) {
	data, ok := m[request.URL.String()]
	if !ok {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Status:     "404 Not Found",
			Body:       io.NopCloser(strings.NewReader("not found")),
			Header:     make(http.Header),
			Request:    request,
		}, nil
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Body:       io.NopCloser(bytes.NewReader(data)),
		Header:     make(http.Header),
		Request:    request,
	}, nil
}

type blockingTransport struct{}

func (blockingTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	<-request.Context().Done()
	return nil, request.Context().Err()
}

func ubuntuTarget(release string) provider.Target {
	return provider.Target{
		ID:         "ubuntu-" + strings.ReplaceAll(release, ".", "") + "-amd64-netboot",
		ProviderID: "ubuntu",
		Name:       "Ubuntu " + release + " amd64 netboot",
		Catalog: provider.CatalogEntry{
			Distribution: "ubuntu",
			Release:      release,
			Architecture: "amd64",
			Kind:         "installer",
		},
	}
}
