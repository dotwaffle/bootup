package linux_test

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
	"github.com/dotwaffle/bootup/internal/providers/linux"
)

func TestProviderPlanResolvesStaticLinuxTarget(t *testing.T) {
	t.Parallel()

	target := linuxTarget()
	p := linux.NewProvider(linux.Config{Targets: []provider.Target{target}})

	plan, err := p.Plan(context.Background(), provider.PlanInput{Target: target})
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

	target := diagnosticTarget()
	p := linux.NewProvider(linux.Config{Targets: []provider.Target{target}})

	plan, err := p.Plan(context.Background(), provider.PlanInput{Target: target})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if plan.Initrd != (provider.Artifact{}) {
		t.Fatalf("initrd = %#v, want none", plan.Initrd)
	}
	if plan.Kernel.Name != "diagnostic-kernel" {
		t.Fatalf("kernel name = %q", plan.Kernel.Name)
	}
}

func TestProviderPlanAppliesSourceArtifactHashPins(t *testing.T) {
	t.Parallel()

	target := linuxTarget()
	target.Source.KernelSHA256 = strings.Repeat("a", 64)
	target.Source.InitrdSHA256 = strings.Repeat("b", 64)
	p := linux.NewProvider(linux.Config{Targets: []provider.Target{target}})

	plan, err := p.Plan(context.Background(), provider.PlanInput{Target: target})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if plan.Kernel.SHA256 != target.Source.KernelSHA256 {
		t.Fatalf("kernel sha256 = %q, want %q", plan.Kernel.SHA256, target.Source.KernelSHA256)
	}
	if plan.Initrd.SHA256 != target.Source.InitrdSHA256 {
		t.Fatalf("initrd sha256 = %q, want %q", plan.Initrd.SHA256, target.Source.InitrdSHA256)
	}
}

func TestFetchAndStageArtifactsAllowsOptionalInitrd(t *testing.T) {
	t.Parallel()

	plan := provider.BootPlan{
		Action: provider.BootActionLinuxKexec,
		Target: provider.Target{ID: "diagnostic-kernel-amd64", ProviderID: "linux"},
		Kernel: provider.Artifact{
			Name: "diagnostic-kernel",
			URL:  "https://boot.example/images/diagnostic-kernel",
		},
	}
	client := &http.Client{Transport: responseMap{
		"https://boot.example/images/diagnostic-kernel": []byte("kernel"),
	}}

	staged, err := linux.FetchAndStageArtifacts(context.Background(), linux.FetchConfig{
		Plan:       plan,
		Client:     client,
		StagingDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("fetch and stage: %v", err)
	}
	if filepath.Base(staged.Kernel.Path) != "diagnostic-kernel" {
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

func TestFetchAndStageArtifactsReportsProgress(t *testing.T) {
	t.Parallel()

	kernelData := []byte("kernel")
	kernelSum := sha256.Sum256(kernelData)
	plan := provider.BootPlan{
		Action: provider.BootActionLinuxKexec,
		Target: provider.Target{ID: "diagnostic-kernel-amd64", ProviderID: "linux"},
		Kernel: provider.Artifact{
			Name:   "diagnostic-kernel",
			URL:    "https://boot.example/images/diagnostic-kernel",
			SHA256: hex.EncodeToString(kernelSum[:]),
		},
	}
	client := &http.Client{Transport: responseMap{
		"https://boot.example/images/diagnostic-kernel": kernelData,
	}}
	var events []provider.StageProgress

	if _, err := linux.FetchAndStageArtifacts(context.Background(), linux.FetchConfig{
		Plan:       plan,
		Client:     client,
		StagingDir: t.TempDir(),
		Progress: func(event provider.StageProgress) error {
			events = append(events, event)
			return nil
		},
	}); err != nil {
		t.Fatalf("fetch and stage: %v", err)
	}

	want := []provider.StageProgress{
		{Operation: provider.StageOperationFetch, State: provider.StageStateStarted, Artifact: "diagnostic-kernel"},
		{Operation: provider.StageOperationFetch, State: provider.StageStateCompleted, Artifact: "diagnostic-kernel"},
		{Operation: provider.StageOperationVerify, State: provider.StageStateStarted, Artifact: "diagnostic-kernel"},
		{Operation: provider.StageOperationVerify, State: provider.StageStateCompleted, Artifact: "diagnostic-kernel"},
		{Operation: provider.StageOperationWrite, State: provider.StageStateStarted, Artifact: "diagnostic-kernel"},
		{Operation: provider.StageOperationWrite, State: provider.StageStateCompleted, Artifact: "diagnostic-kernel"},
	}
	if !equalStageProgress(events, want) {
		t.Fatalf("progress events = %#v, want %#v", events, want)
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

func equalStageProgress(a []provider.StageProgress, b []provider.StageProgress) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
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

func diagnosticTarget() provider.Target {
	return provider.Target{
		ID:         "diagnostic-kernel-amd64",
		ProviderID: "linux",
		Name:       "Diagnostic kernel amd64",
		Catalog: provider.CatalogEntry{
			Distribution: "diagnostic",
			Release:      "latest",
			Architecture: "amd64",
			Kind:         "tool",
		},
		Source: provider.SourceEntry{
			BaseURL:    "https://boot.example",
			KernelPath: "images/diagnostic-kernel",
		},
	}
}
