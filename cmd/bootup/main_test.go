package main

import (
	"bytes"
	"context"
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

func TestRunAppliesAppendCmdlineFlag(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	err := runWithIO(context.Background(), []string{
		"--mode", "plan-target",
		"--target", "memtest86plus-800-amd64",
		"--append-cmdline", "console=ttyS1",
	}, strings.NewReader(""), &stdout, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(stdout.String(), "cmdline\tconsole=ttyS1") {
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
