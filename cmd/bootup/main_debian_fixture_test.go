//go:build bootup_debian_fixture

package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunDiscoversTargetsWithFixtureProvider(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	if err := runWithIO(context.Background(), []string{
		"--mode", "discover-targets",
		"--discovery-family", "debian",
	}, strings.NewReader(""), &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("run discovery mode: %v", err)
	}

	if !strings.Contains(stdout.String(), "debian-trixie-amd64-netboot") {
		t.Fatalf("stdout = %q, want discovered Debian trixie target", stdout.String())
	}
}
