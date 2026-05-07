package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestRunRejectsMissingProviderConfig(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "missing.json")
	err := run(context.Background(), []string{"--provider-config", configPath})
	if err == nil {
		t.Fatal("run succeeded, want provider config error")
	}
	if !strings.Contains(err.Error(), "load provider config") {
		t.Fatalf("run error = %q, want provider config context", err)
	}
}

func TestRunRejectsMissingCatalog(t *testing.T) {
	t.Parallel()

	catalogPath := filepath.Join(t.TempDir(), "missing.json")
	err := run(context.Background(), []string{"--catalog", catalogPath})
	if err == nil {
		t.Fatal("run succeeded, want catalog error")
	}
	if !strings.Contains(err.Error(), "load catalog") {
		t.Fatalf("run error = %q, want catalog context", err)
	}
}

func TestRunRejectsLocalAndHostedCatalogSources(t *testing.T) {
	t.Parallel()

	err := run(context.Background(), []string{
		"--catalog", filepath.Join(t.TempDir(), "catalog.json"),
		"--catalog-url", "https://catalog.example/bootup/catalog.json",
	})
	if err == nil {
		t.Fatal("run succeeded, want catalog source error")
	}
	if !strings.Contains(err.Error(), "catalog") || !strings.Contains(err.Error(), "catalog-url") {
		t.Fatalf("run error = %q, want catalog source context", err)
	}
}

func TestRunRejectsHostedCatalogWithoutTrust(t *testing.T) {
	t.Parallel()

	err := run(context.Background(), []string{
		"--catalog-url", "https://catalog.example/bootup/catalog.json",
	})
	if err == nil {
		t.Fatal("run succeeded, want hosted trust error")
	}
	if !strings.Contains(err.Error(), "trust") {
		t.Fatalf("run error = %q, want trust context", err)
	}
}

func TestRunLoadsHostedCatalogFromCacheFallback(t *testing.T) {
	t.Parallel()

	data := []byte(`{
		"schema_version": 1,
		"targets": [{
			"id": "debian-trixie-amd64-netboot",
			"provider_id": "debian",
			"name": "Debian trixie amd64 netboot",
			"catalog": {
				"distribution": "debian",
				"release": "trixie",
				"architecture": "amd64",
				"kind": "installer"
			}
		}]
	}`)
	cachePath := filepath.Join(t.TempDir(), "catalog-cache.json")
	if err := os.WriteFile(cachePath, data, 0o644); err != nil {
		t.Fatalf("write cache: %v", err)
	}
	sum := sha256.Sum256(data)

	var stdout bytes.Buffer
	err := runWithIO(context.Background(), []string{
		"--mode", "list-targets",
		"--catalog-url", "https://127.0.0.1:1/catalog.json",
		"--catalog-sha256", hex.EncodeToString(sum[:]),
		"--catalog-cache", cachePath,
		"--catalog-cache-fallback",
	}, strings.NewReader(""), &stdout, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(stdout.String(), "debian-trixie-amd64-netboot") {
		t.Fatalf("stdout = %q, want hosted cache target", stdout.String())
	}
}

func TestRunAcceptsDiscoverTargetsModeFlag(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	err := runWithIO(context.Background(), []string{"--mode", "discover-targets"}, strings.NewReader(""), &stdout, &bytes.Buffer{})
	if err == nil {
		t.Fatal("run succeeded, want missing discovery family error")
	}
	if !strings.Contains(err.Error(), "discovery family is required") {
		t.Fatalf("run error = %v, want discovery family requirement", err)
	}
}

func TestRunAcceptsCatalogMatrixModeFlag(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	err := runWithIO(context.Background(), []string{"--mode", "catalog-matrix"}, strings.NewReader(""), &stdout, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{
		"bootup catalog matrix",
		"target\tprovider\taction\tplan\ttrust\tsmoke\terror",
		"opensuse-leap-160-amd64-netboot",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("stdout = %q, want %q", got, want)
		}
	}
}

func TestRunAppliesAppendCmdlineFlag(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	err := runWithIO(context.Background(), []string{
		"--mode", "plan-target",
		"--target", "opensuse-leap-160-amd64-netboot",
		"--append-cmdline", "console=ttyS1",
	}, strings.NewReader(""), &stdout, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(stdout.String(), "console=ttyS0 console=ttyS1") {
		t.Fatalf("stdout = %q, want appended cmdline", stdout.String())
	}
}

func TestRunAppliesTargetOptionFlagsBeforeAppendCmdline(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	err := runWithIO(context.Background(), []string{
		"--mode", "plan-target",
		"--target", "opensuse-leap-160-amd64-netboot",
		"--option", "mirror-url=https://mirror.example/opensuse",
		"--option", "text-install=true",
		"--append-cmdline", "console=ttyS1",
	}, strings.NewReader(""), &stdout, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	want := "cmdline\tnetsetup=dhcp install=https://download.opensuse.org/distribution/leap/16.0/repo/oss console=ttyS0 textmode=1 install=https://mirror.example/opensuse console=ttyS1"
	if !strings.Contains(stdout.String(), want) {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
}

func TestParseDNSServers(t *testing.T) {
	t.Parallel()

	got := parseDNSServers("192.0.2.53, 192.0.2.54")
	want := []string{"192.0.2.53", "192.0.2.54"}
	if !slices.Equal(got, want) {
		t.Fatalf("DNS servers = %#v, want %#v", got, want)
	}
	if got := parseDNSServers(" "); len(got) != 0 {
		t.Fatalf("empty DNS servers = %#v, want none", got)
	}
}
