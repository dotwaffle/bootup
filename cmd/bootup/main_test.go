package main

import (
	"bytes"
	"context"
	"path/filepath"
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
