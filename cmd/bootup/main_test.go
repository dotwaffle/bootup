package main

import (
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
