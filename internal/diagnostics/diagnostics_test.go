package diagnostics_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/dotwaffle/bootup/internal/diagnostics"
	"github.com/dotwaffle/bootup/internal/provider"
)

func TestWriteBundleWritesSummaryAndStreams(t *testing.T) {
	t.Parallel()

	selected := []provider.SelectedOption{
		{ID: "mirror-url", Value: "https://secret.example/install token"},
		{ID: "text-install", Value: "true"},
	}
	summary := diagnostics.BuildSummary(diagnostics.SummaryInput{
		Mode:              "plan-target",
		TargetID:          "opensuse-leap-160-amd64-netboot",
		DiscoveryFamilyID: "fedora",
		SelectedOptions:   selected,
		SecretInputIDs:    []string{"installer-password"},
		SecretRefIDs:      []string{"installer-password"},
		Catalog: diagnostics.CatalogPosture{
			Source:         "hosted",
			SHA256:         true,
			Ed25519:        true,
			Freshness:      true,
			Cache:          true,
			CacheFallback:  true,
			MaxBytes:       true,
			LocalPathSet:   false,
			HostedURLSet:   true,
			SignatureFiles: true,
		},
		ProviderConfig: diagnostics.ProviderConfigPosture{PathSet: true},
		Error:          errors.New(`invalid target option: target opensuse option mirror-url value "https://secret.example/install token" is invalid; read secret input installer-password /run/bootup/secrets/installer-password: permission denied`),
		RedactValues:   []string{"/run/bootup/secrets/installer-password"},
	})

	bundleDir, err := diagnostics.WriteBundle(diagnostics.Bundle{
		RootDir:   t.TempDir(),
		CreatedAt: time.Date(2026, 5, 7, 9, 10, 11, 0, time.UTC),
		Summary:   summary,
		Stdout:    []byte("[planning] openSUSE Leap 16.0 amd64 installer\n"),
		Stderr:    []byte("time=2026-05-07T09:10:11Z level=INFO msg=\"bootup started\"\n"),
	})
	if err != nil {
		t.Fatalf("write bundle: %v", err)
	}

	if filepath.Base(bundleDir) != "bootup-20260507T091011Z" {
		t.Fatalf("bundle dir = %q, want timestamped bootup dir", bundleDir)
	}
	assertFileMode(t, bundleDir, 0o700)

	summaryBytes := readFile(t, filepath.Join(bundleDir, "summary.json"))
	if strings.Contains(string(summaryBytes), "https://secret.example/install token") {
		t.Fatalf("summary contains selected option value: %s", summaryBytes)
	}
	if strings.Contains(string(summaryBytes), "token") {
		t.Fatalf("summary contains token text: %s", summaryBytes)
	}
	if strings.Contains(string(summaryBytes), "/run/bootup/secrets") {
		t.Fatalf("summary contains secret input path: %s", summaryBytes)
	}

	var got diagnostics.Summary
	if err := json.Unmarshal(summaryBytes, &got); err != nil {
		t.Fatalf("decode summary: %v", err)
	}
	if got.SchemaVersion != 1 {
		t.Fatalf("schema version = %d, want 1", got.SchemaVersion)
	}
	if got.CreatedAt != "2026-05-07T09:10:11Z" {
		t.Fatalf("created_at = %q, want fixed timestamp", got.CreatedAt)
	}
	if !slices.Equal(got.SelectedOptionIDs, []string{"mirror-url", "text-install"}) {
		t.Fatalf("selected option IDs = %#v, want IDs only", got.SelectedOptionIDs)
	}
	if !slices.Equal(got.SecretInputIDs, []string{"installer-password"}) {
		t.Fatalf("secret input IDs = %#v, want installer-password", got.SecretInputIDs)
	}
	if !slices.Equal(got.SecretRefIDs, []string{"installer-password"}) {
		t.Fatalf("secret ref IDs = %#v, want installer-password", got.SecretRefIDs)
	}
	if !strings.Contains(got.Error, "<redacted>") {
		t.Fatalf("summary error = %q, want redaction marker", got.Error)
	}
	if got.Catalog.Source != "hosted" || !got.Catalog.SHA256 || !got.Catalog.Ed25519 {
		t.Fatalf("catalog posture = %#v, want hosted authenticated posture", got.Catalog)
	}
	if !got.ProviderConfig.PathSet {
		t.Fatalf("provider config posture = %#v, want path presence", got.ProviderConfig)
	}

	stdout := readFile(t, filepath.Join(bundleDir, "stdout.txt"))
	if string(stdout) != "[planning] openSUSE Leap 16.0 amd64 installer\n" {
		t.Fatalf("stdout = %q, want captured stdout", stdout)
	}
	stderr := readFile(t, filepath.Join(bundleDir, "stderr.txt"))
	if !strings.Contains(string(stderr), "bootup started") {
		t.Fatalf("stderr = %q, want captured log output", stderr)
	}
	assertFileMode(t, filepath.Join(bundleDir, "summary.json"), 0o600)
	assertFileMode(t, filepath.Join(bundleDir, "stdout.txt"), 0o600)
	assertFileMode(t, filepath.Join(bundleDir, "stderr.txt"), 0o600)
}

func TestWriteBundleReportsCreateFailure(t *testing.T) {
	t.Parallel()

	rootFile := filepath.Join(t.TempDir(), "diagnostics")
	if err := os.WriteFile(rootFile, []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("write root file: %v", err)
	}

	_, err := diagnostics.WriteBundle(diagnostics.Bundle{
		RootDir:   rootFile,
		CreatedAt: time.Date(2026, 5, 7, 9, 10, 11, 0, time.UTC),
		Summary:   diagnostics.BuildSummary(diagnostics.SummaryInput{Error: errors.New("boot failed")}),
	})
	if err == nil {
		t.Fatal("write bundle succeeded, want create failure")
	}
	if !strings.Contains(err.Error(), "create diagnostics directory") {
		t.Fatalf("write error = %q, want create context", err)
	}
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}

func assertFileMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("%s mode = %o, want %o", path, got, want)
	}
}
