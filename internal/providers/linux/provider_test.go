package linux_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dotwaffle/bootup/internal/provider"
	"github.com/dotwaffle/bootup/internal/providers/linux"
)

func TestProviderPlanResolvesStaticLinuxTarget(t *testing.T) {
	t.Parallel()

	target := linuxTarget()
	p := linux.NewProvider(linux.Config{Targets: []provider.Target{target}})

	plan, err := p.Plan(context.Background(), target)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if plan.Action != provider.BootActionLinuxKexec {
		t.Fatalf("plan action = %q, want linux-kexec", plan.Action)
	}
	if plan.Kernel.URL != "https://download.example/opensuse/boot/x86_64/loader/linux" {
		t.Fatalf("kernel URL = %q", plan.Kernel.URL)
	}
	if plan.Initrd.URL != "https://download.example/opensuse/boot/x86_64/loader/initrd" {
		t.Fatalf("initrd URL = %q", plan.Initrd.URL)
	}
	if plan.Cmdline != "netsetup=dhcp install=https://download.example/opensuse console=ttyS0" {
		t.Fatalf("cmdline = %q", plan.Cmdline)
	}
}

func TestProviderPlanAllowsKernelOnlyTarget(t *testing.T) {
	t.Parallel()

	target := memtestTarget()
	p := linux.NewProvider(linux.Config{Targets: []provider.Target{target}})

	plan, err := p.Plan(context.Background(), target)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if plan.Initrd != (provider.Artifact{}) {
		t.Fatalf("initrd = %#v, want none", plan.Initrd)
	}
	if plan.Kernel.Name != "mt86p_800_x86_64" {
		t.Fatalf("kernel name = %q", plan.Kernel.Name)
	}
}

func TestFetchAndStageArtifactsAllowsOptionalInitrd(t *testing.T) {
	t.Parallel()

	plan := provider.BootPlan{
		Action: provider.BootActionLinuxKexec,
		Target: provider.Target{ID: "memtest86plus-800-amd64", ProviderID: "linux"},
		Kernel: provider.Artifact{
			Name: "mt86p_800_x86_64",
			URL:  "https://boot.example/images/mt86plus/800/mt86p_800_x86_64",
		},
	}
	client := &http.Client{Transport: responseMap{
		"https://boot.example/images/mt86plus/800/mt86p_800_x86_64": []byte("kernel"),
	}}

	staged, err := linux.FetchAndStageArtifacts(context.Background(), linux.FetchConfig{
		Plan:       plan,
		Client:     client,
		StagingDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("fetch and stage: %v", err)
	}
	if filepath.Base(staged.Kernel.Path) != "mt86p_800_x86_64" {
		t.Fatalf("kernel path = %q", staged.Kernel.Path)
	}
	if staged.Initrd.Path != "" {
		t.Fatalf("initrd path = %q, want none", staged.Initrd.Path)
	}
	got, err := os.ReadFile(staged.Kernel.Path)
	if err != nil {
		t.Fatalf("read staged kernel: %v", err)
	}
	if string(got) != "kernel" {
		t.Fatalf("staged kernel = %q", got)
	}
}

func TestFetchAndStageArtifactsRejectsUnverifiedHTTP(t *testing.T) {
	t.Parallel()

	_, err := linux.FetchAndStageArtifacts(context.Background(), linux.FetchConfig{
		Plan: provider.BootPlan{
			Action: provider.BootActionLinuxKexec,
			Target: provider.Target{ID: "opensuse-leap-160-amd64-netboot", ProviderID: "linux"},
			Kernel: provider.Artifact{Name: "linux", URL: "http://download.example/linux"},
			Initrd: provider.Artifact{Name: "initrd", URL: "http://download.example/initrd"},
		},
		StagingDir: t.TempDir(),
	})
	if err == nil {
		t.Fatal("fetch and stage succeeded, want https failure")
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

func linuxTarget() provider.Target {
	return provider.Target{
		ID:         "opensuse-leap-160-amd64-netboot",
		ProviderID: "linux",
		Name:       "openSUSE Leap 16.0 amd64 installer",
		Catalog: provider.CatalogEntry{
			Distribution: "opensuse",
			Release:      "leap-16.0",
			Architecture: "amd64",
			Kind:         "installer",
		},
		Source: provider.SourceEntry{
			BaseURL:    "https://download.example/opensuse",
			KernelPath: "boot/x86_64/loader/linux",
			InitrdPath: "boot/x86_64/loader/initrd",
			Cmdline:    "netsetup=dhcp install={base_url} console=ttyS0",
		},
	}
}

func memtestTarget() provider.Target {
	return provider.Target{
		ID:         "memtest86plus-800-amd64",
		ProviderID: "linux",
		Name:       "MemTest86+ 8.00 amd64",
		Catalog: provider.CatalogEntry{
			Distribution: "memtest86plus",
			Release:      "8.00",
			Architecture: "amd64",
			Kind:         "tool",
		},
		Source: provider.SourceEntry{
			BaseURL:    "https://boot.example",
			KernelPath: "images/mt86plus/800/mt86p_800_x86_64",
		},
	}
}
