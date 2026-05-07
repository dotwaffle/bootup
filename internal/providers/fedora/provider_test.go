package fedora_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dotwaffle/bootup/internal/provider"
	"github.com/dotwaffle/bootup/internal/providers/fedora"
)

func TestProviderTargetsFedora44AMD64Server(t *testing.T) {
	t.Parallel()

	targets, err := fedora.NewProvider(fedora.Config{}).Targets(context.Background())
	if err != nil {
		t.Fatalf("targets: %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("targets length = %d, want 1", len(targets))
	}
	target := targets[0]
	if target.ID != "fedora-44-amd64-server-netboot" {
		t.Fatalf("target ID = %q, want Fedora 44", target.ID)
	}
	if target.ProviderID != "fedora" {
		t.Fatalf("provider ID = %q, want fedora", target.ProviderID)
	}
	if target.Catalog.Release != "44" || target.Catalog.Architecture != "amd64" || target.Catalog.Kind != "installer" {
		t.Fatalf("catalog = %#v, want Fedora 44 amd64 installer", target.Catalog)
	}
}

func TestProviderPlanResolvesServerNetbootURLs(t *testing.T) {
	t.Parallel()

	p := fedora.NewProvider(fedora.Config{
		ReleaseURL:   "https://mirror.example/fedora/releases/44/Server/x86_64/os",
		KernelSHA256: strings.Repeat("a", 64),
		InitrdSHA256: strings.Repeat("b", 64),
	})

	plan, err := p.Plan(context.Background(), provider.PlanInput{
		Target: provider.Target{
			ID:         "fedora-44-amd64-server-netboot",
			ProviderID: "fedora",
		},
	})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}

	if plan.Kernel.URL != "https://mirror.example/fedora/releases/44/Server/x86_64/os/images/pxeboot/vmlinuz" {
		t.Fatalf("kernel URL = %q", plan.Kernel.URL)
	}
	if plan.Initrd.URL != "https://mirror.example/fedora/releases/44/Server/x86_64/os/images/pxeboot/initrd.img" {
		t.Fatalf("initrd URL = %q", plan.Initrd.URL)
	}
	if plan.Cmdline != "inst.repo=https://mirror.example/fedora/releases/44/Server/x86_64/os ip=dhcp console=ttyS0" {
		t.Fatalf("cmdline = %q", plan.Cmdline)
	}
	if plan.Kernel.SHA256 != strings.Repeat("a", 64) || plan.Initrd.SHA256 != strings.Repeat("b", 64) {
		t.Fatalf("hash pins = %q/%q", plan.Kernel.SHA256, plan.Initrd.SHA256)
	}
}

func TestProviderPlanUsesTargetSourceBaseURL(t *testing.T) {
	t.Parallel()

	target := fedoraTarget("43")
	target.Source.BaseURL = "https://download.example/fedora/releases/43/Server/x86_64/os/"
	p := fedora.NewProvider(fedora.Config{
		Targets: []provider.Target{target},
		Client: &http.Client{Transport: responseMap{
			"https://download.example/fedora/releases/43/Server/x86_64/os/.treeinfo": []byte(fedoraTreeinfo(strings.Repeat("a", 64), strings.Repeat("b", 64))),
		}},
	})

	plan, err := p.Plan(context.Background(), provider.PlanInput{Target: target})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if plan.Kernel.URL != "https://download.example/fedora/releases/43/Server/x86_64/os/images/pxeboot/vmlinuz" {
		t.Fatalf("kernel URL = %q", plan.Kernel.URL)
	}
}

func TestProviderPlanUsesTreeinfoChecksumsByDefault(t *testing.T) {
	t.Parallel()

	kernelSHA256 := strings.Repeat("a", 64)
	initrdSHA256 := strings.Repeat("b", 64)
	p := fedora.NewProvider(fedora.Config{
		ReleaseURL: "https://mirror.example/fedora/releases/44/Server/x86_64/os",
		Client: &http.Client{Transport: responseMap{
			"https://mirror.example/fedora/releases/44/Server/x86_64/os/.treeinfo": []byte(fedoraTreeinfo(kernelSHA256, initrdSHA256)),
		}},
	})

	plan, err := p.Plan(context.Background(), provider.PlanInput{
		Target: provider.Target{
			ID:         "fedora-44-amd64-server-netboot",
			ProviderID: "fedora",
		},
	})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if plan.Kernel.SHA256 != kernelSHA256 {
		t.Fatalf("kernel SHA-256 = %q, want treeinfo hash", plan.Kernel.SHA256)
	}
	if plan.Initrd.SHA256 != initrdSHA256 {
		t.Fatalf("initrd SHA-256 = %q, want treeinfo hash", plan.Initrd.SHA256)
	}
}

func TestProviderPlanFailsWithoutTreeinfoChecksums(t *testing.T) {
	t.Parallel()

	p := fedora.NewProvider(fedora.Config{
		ReleaseURL: "https://mirror.example/fedora/releases/44/Server/x86_64/os",
		Client: &http.Client{Transport: responseMap{
			"https://mirror.example/fedora/releases/44/Server/x86_64/os/.treeinfo": []byte(`[checksums]
images/pxeboot/vmlinuz = sha256:` + strings.Repeat("a", 64) + `
`),
		}},
	})

	_, err := p.Plan(context.Background(), provider.PlanInput{
		Target: provider.Target{
			ID:         "fedora-44-amd64-server-netboot",
			ProviderID: "fedora",
		},
	})
	if err == nil {
		t.Fatal("plan succeeded, want missing initrd checksum error")
	}
	if !strings.Contains(err.Error(), ".treeinfo") || !strings.Contains(err.Error(), "images/pxeboot/initrd.img") {
		t.Fatalf("plan error = %q, want treeinfo initrd checksum context", err)
	}
}

func TestProviderPlanOfflineRefusesTreeinfoFallback(t *testing.T) {
	t.Parallel()

	p := fedora.NewProvider(fedora.Config{
		ReleaseURL: "https://mirror.example/fedora/releases/44/Server/x86_64/os",
		Client: &http.Client{Transport: responseMap{
			"https://mirror.example/fedora/releases/44/Server/x86_64/os/.treeinfo": []byte(fedoraTreeinfo(strings.Repeat("a", 64), strings.Repeat("b", 64))),
		}},
	})

	_, err := p.Plan(context.Background(), provider.PlanInput{
		Target: provider.Target{
			ID:         "fedora-44-amd64-server-netboot",
			ProviderID: "fedora",
		},
		Offline: true,
	})
	if err == nil {
		t.Fatal("plan succeeded, want offline metadata error")
	}
	if !strings.Contains(err.Error(), "offline") || !strings.Contains(err.Error(), ".treeinfo") {
		t.Fatalf("plan error = %q, want offline treeinfo context", err)
	}
}

func TestProviderPlanUsesExplicitPinsWithoutTreeinfo(t *testing.T) {
	t.Parallel()

	kernelSHA256 := strings.Repeat("a", 64)
	initrdSHA256 := strings.Repeat("b", 64)
	p := fedora.NewProvider(fedora.Config{
		ReleaseURL:   "https://mirror.example/fedora/releases/44/Server/x86_64/os",
		KernelSHA256: kernelSHA256,
		InitrdSHA256: initrdSHA256,
		Client:       &http.Client{Transport: responseMap{}},
	})

	plan, err := p.Plan(context.Background(), provider.PlanInput{
		Target: provider.Target{
			ID:         "fedora-44-amd64-server-netboot",
			ProviderID: "fedora",
		},
	})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if plan.Kernel.SHA256 != kernelSHA256 || plan.Initrd.SHA256 != initrdSHA256 {
		t.Fatalf("plan hashes = %q/%q, want explicit pins", plan.Kernel.SHA256, plan.Initrd.SHA256)
	}
}

func TestProviderPlanUsesTargetSourcePinsWithoutTreeinfo(t *testing.T) {
	t.Parallel()

	target := fedoraTarget("44")
	target.Source = provider.SourceEntry{
		BaseURL:      "https://mirror.example/fedora/releases/44/Server/x86_64/os",
		KernelPath:   "images/pxeboot/vmlinuz",
		InitrdPath:   "images/pxeboot/initrd.img",
		KernelSHA256: strings.Repeat("a", 64),
		InitrdSHA256: strings.Repeat("b", 64),
	}
	p := fedora.NewProvider(fedora.Config{
		Targets: []provider.Target{target},
		Client:  &http.Client{Transport: responseMap{}},
	})

	plan, err := p.Plan(context.Background(), provider.PlanInput{Target: target})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if plan.Kernel.SHA256 != target.Source.KernelSHA256 || plan.Initrd.SHA256 != target.Source.InitrdSHA256 {
		t.Fatalf("plan hashes = %q/%q, want target source pins", plan.Kernel.SHA256, plan.Initrd.SHA256)
	}
}

func TestProviderDiscoveryFamily(t *testing.T) {
	t.Parallel()

	family := fedora.NewProvider(fedora.Config{}).DiscoveryFamily()
	if family.ID != "fedora" || family.ProviderID != "fedora" {
		t.Fatalf("family IDs = %q/%q, want fedora/fedora", family.ID, family.ProviderID)
	}
	if family.Name != "Fedora" {
		t.Fatalf("family name = %q, want Fedora", family.Name)
	}
}

func TestProviderDiscoversServerNetbootTargets(t *testing.T) {
	t.Parallel()

	p := fedora.NewProvider(fedora.Config{
		DiscoveryURL: "https://mirror.example/fedora/releases",
		Client: &http.Client{Transport: responseMap{
			"https://mirror.example/fedora/releases/": []byte(`
				<a href="44/">44/</a>
				<a href="45/">45/</a>
				<a href="rawhide/">rawhide/</a>
			`),
			"https://mirror.example/fedora/releases/44/Server/x86_64/os/images/pxeboot/vmlinuz":    []byte("kernel"),
			"https://mirror.example/fedora/releases/44/Server/x86_64/os/images/pxeboot/initrd.img": []byte("initrd"),
			"https://mirror.example/fedora/releases/45/Server/x86_64/os/images/pxeboot/vmlinuz":    []byte("kernel"),
		}},
	})

	targets, err := p.DiscoverTargets(context.Background())
	if err != nil {
		t.Fatalf("discover targets: %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("targets = %#v, want one complete Fedora release", targets)
	}
	target := targets[0]
	if target.ID != "fedora-44-amd64-server-netboot" {
		t.Fatalf("target ID = %q, want Fedora 44", target.ID)
	}
	if target.Source.BaseURL != "https://mirror.example/fedora/releases/44/Server/x86_64/os" {
		t.Fatalf("source base URL = %q, want Fedora 44 install tree", target.Source.BaseURL)
	}
	if target.Catalog.Release != "44" || target.Catalog.Architecture != "amd64" || target.Catalog.Kind != "installer" {
		t.Fatalf("catalog = %#v, want Fedora 44 amd64 installer", target.Catalog)
	}
}

func TestFetchAndStageArtifactsAllowsHTTPSOnlyNetboot(t *testing.T) {
	t.Parallel()

	kernel := []byte("kernel")
	initrd := []byte("initrd")
	plan := provider.BootPlan{
		Target: provider.Target{ID: "fedora-44-amd64-server-netboot", ProviderID: "fedora"},
		Kernel: provider.Artifact{
			Name: "vmlinuz",
			URL:  "https://mirror.example/fedora/releases/44/Server/x86_64/os/images/pxeboot/vmlinuz",
		},
		Initrd: provider.Artifact{
			Name: "initrd.img",
			URL:  "https://mirror.example/fedora/releases/44/Server/x86_64/os/images/pxeboot/initrd.img",
		},
	}
	client := &http.Client{Transport: responseMap{
		"https://mirror.example/fedora/releases/44/Server/x86_64/os/images/pxeboot/vmlinuz":    kernel,
		"https://mirror.example/fedora/releases/44/Server/x86_64/os/images/pxeboot/initrd.img": initrd,
	}}

	staged, err := fedora.FetchAndStageArtifacts(context.Background(), fedora.FetchConfig{
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
	if filepath.Base(staged.Initrd.Path) != "initrd.img" {
		t.Fatalf("staged initrd path = %q, want initrd.img", staged.Initrd.Path)
	}
}

func TestFetchAndStageArtifactsRejectsUnverifiedHTTP(t *testing.T) {
	t.Parallel()

	plan := provider.BootPlan{
		Target: provider.Target{ID: "fedora-44-amd64-server-netboot", ProviderID: "fedora"},
		Kernel: provider.Artifact{Name: "vmlinuz", URL: "http://mirror.example/fedora/vmlinuz"},
		Initrd: provider.Artifact{Name: "initrd.img", URL: "http://mirror.example/fedora/initrd.img"},
	}

	_, err := fedora.FetchAndStageArtifacts(context.Background(), fedora.FetchConfig{
		Plan:       plan,
		StagingDir: t.TempDir(),
	})
	if err == nil {
		t.Fatal("fetch and stage succeeded, want https failure")
	}
}

func TestFetchAndStageArtifactsVerifiesHashPins(t *testing.T) {
	t.Parallel()

	kernel := []byte("kernel")
	initrd := []byte("initrd")
	kernelSum := sha256.Sum256(kernel)
	initrdSum := sha256.Sum256(initrd)
	plan := provider.BootPlan{
		Target: provider.Target{ID: "fedora-44-amd64-server-netboot", ProviderID: "fedora"},
		Kernel: provider.Artifact{
			Name:   "vmlinuz",
			URL:    "https://mirror.example/fedora/vmlinuz",
			SHA256: hex.EncodeToString(kernelSum[:]),
		},
		Initrd: provider.Artifact{
			Name:   "initrd.img",
			URL:    "https://mirror.example/fedora/initrd.img",
			SHA256: hex.EncodeToString(initrdSum[:]),
		},
	}
	client := &http.Client{Transport: responseMap{
		"https://mirror.example/fedora/vmlinuz":    kernel,
		"https://mirror.example/fedora/initrd.img": initrd,
	}}

	if _, err := fedora.FetchAndStageArtifacts(context.Background(), fedora.FetchConfig{
		Plan:       plan,
		Client:     client,
		StagingDir: t.TempDir(),
	}); err != nil {
		t.Fatalf("fetch and stage: %v", err)
	}
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

func fedoraTarget(release string) provider.Target {
	return provider.Target{
		ID:         "fedora-" + release + "-amd64-server-netboot",
		ProviderID: "fedora",
		Name:       "Fedora Server " + release + " amd64 netboot",
		Catalog: provider.CatalogEntry{
			Distribution: "fedora",
			Release:      release,
			Architecture: "amd64",
			Kind:         "installer",
		},
	}
}

func fedoraTreeinfo(kernelSHA256 string, initrdSHA256 string) string {
	return `[checksums]
images/pxeboot/initrd.img = sha256:` + initrdSHA256 + `
images/pxeboot/vmlinuz = sha256:` + kernelSHA256 + `

[images-x86_64]
initrd = images/pxeboot/initrd.img
kernel = images/pxeboot/vmlinuz
`
}
