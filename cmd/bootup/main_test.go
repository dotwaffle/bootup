package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/dotwaffle/bootup/internal/diagnostics"
	"github.com/dotwaffle/bootup/internal/policy"
	"github.com/dotwaffle/bootup/internal/provider"
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

func TestRunVersionBypassesStartup(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runWithIO(context.Background(), []string{
		"--version",
		"--provider-config", filepath.Join(t.TempDir(), "missing.json"),
		"--catalog", filepath.Join(t.TempDir(), "missing-catalog.json"),
	}, strings.NewReader(""), &stdout, &stderr)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty output", stderr.String())
	}
	got := stdout.String()
	for _, want := range []string{
		"bootup version\n",
		"version\tdevel\n",
		"commit\tunknown\n",
		"date\tunknown\n",
		"dirty\tunknown\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("stdout = %q, want %q", got, want)
		}
	}
	if !strings.Contains(got, "go\t") {
		t.Fatalf("stdout = %q, want Go runtime field", got)
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

func TestRunDiscoversFedoraTargets(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			_, _ = w.Write([]byte(`<a href="44/">44/</a>`))
		case "/44/Server/x86_64/os/images/pxeboot/vmlinuz",
			"/44/Server/x86_64/os/images/pxeboot/initrd.img":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	configPath := filepath.Join(t.TempDir(), "providers.json")
	if err := os.WriteFile(configPath, []byte(`{
		"providers": {
			"fedora": {
				"discovery_url": "`+server.URL+`"
			}
		}
	}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var stdout bytes.Buffer
	err := runWithIO(context.Background(), []string{
		"--mode", "discover-targets",
		"--discovery-family", "fedora",
		"--provider-config", configPath,
	}, strings.NewReader(""), &stdout, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(stdout.String(), "fedora-44-amd64-server-netboot") {
		t.Fatalf("stdout = %q, want Fedora discovered target", stdout.String())
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
		"target\tprovider\tdistribution\trelease\tarchitecture\tkind\tlifecycle\taction\tplan\ttrust\tsmoke\terror",
		"opensuse-leap-160-amd64-netboot",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("stdout = %q, want %q", got, want)
		}
	}
}

func TestRunCatalogMatrixReportsHashPinnedLocalCatalog(t *testing.T) {
	t.Parallel()

	catalogPath := filepath.Join(t.TempDir(), "catalog.json")
	if err := os.WriteFile(catalogPath, []byte(`{
		"schema_version": 1,
		"targets": [{
			"id": "opensuse-leap-160-amd64-netboot",
			"provider_id": "linux",
			"name": "openSUSE Leap 16.0 amd64 installer",
			"catalog": {
				"distribution": "opensuse",
				"release": "leap-16.0",
				"architecture": "amd64",
				"kind": "installer"
			},
			"source": {
				"base_url": "https://download.example/opensuse",
				"kernel_path": "boot/x86_64/loader/linux",
				"initrd_path": "boot/x86_64/loader/initrd",
				"kernel_sha256": "`+strings.Repeat("a", 64)+`",
				"initrd_sha256": "`+strings.Repeat("b", 64)+`"
			}
		}]
	}`), 0o644); err != nil {
		t.Fatalf("write catalog: %v", err)
	}

	var stdout bytes.Buffer
	err := runWithIO(context.Background(), []string{
		"--mode", "catalog-matrix",
		"--catalog", catalogPath,
	}, strings.NewReader(""), &stdout, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	want := "opensuse-leap-160-amd64-netboot\tlinux\topensuse\tleap-16.0\tamd64\tinstaller\t\tlinux-kexec\tok\thash-pinned\tlive-stage,catalog-qemu\t"
	if !strings.Contains(stdout.String(), want) {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
}

func TestRunCatalogIncludeDefaultComposesLocalCatalog(t *testing.T) {
	t.Parallel()

	catalogPath := writeLinuxCatalog(t, "opensuse-lab-amd64-netboot")

	var stdout bytes.Buffer
	err := runWithIO(context.Background(), []string{
		"--mode", "list-targets",
		"--catalog", catalogPath,
		"--catalog-include-default",
	}, strings.NewReader(""), &stdout, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(stdout.String(), "opensuse-lab-amd64-netboot") {
		t.Fatalf("stdout = %q, want local catalog target", stdout.String())
	}
	if !strings.Contains(stdout.String(), "debian-trixie-amd64-netboot") {
		t.Fatalf("stdout = %q, want embedded default target", stdout.String())
	}
}

func TestRunLocalCatalogStillReplacesDefaultByDefault(t *testing.T) {
	t.Parallel()

	catalogPath := writeLinuxCatalog(t, "opensuse-lab-amd64-netboot")

	var stdout bytes.Buffer
	err := runWithIO(context.Background(), []string{
		"--mode", "list-targets",
		"--catalog", catalogPath,
	}, strings.NewReader(""), &stdout, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(stdout.String(), "opensuse-lab-amd64-netboot") {
		t.Fatalf("stdout = %q, want local catalog target", stdout.String())
	}
	if strings.Contains(stdout.String(), "debian-trixie-amd64-netboot") {
		t.Fatalf("stdout = %q, did not want embedded default target", stdout.String())
	}
}

func TestRunCatalogIncludeDefaultRejectsDuplicateTargetID(t *testing.T) {
	t.Parallel()

	catalogPath := filepath.Join(t.TempDir(), "catalog.json")
	if err := os.WriteFile(catalogPath, []byte(`{
		"schema_version": 1,
		"targets": [{
			"id": "debian-trixie-amd64-netboot",
			"provider_id": "debian",
			"name": "Duplicate Debian trixie",
			"catalog": {
				"distribution": "debian",
				"release": "trixie",
				"architecture": "amd64",
				"kind": "installer"
			}
		}]
	}`), 0o644); err != nil {
		t.Fatalf("write catalog: %v", err)
	}

	err := runWithIO(context.Background(), []string{
		"--mode", "list-targets",
		"--catalog", catalogPath,
		"--catalog-include-default",
	}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("run succeeded, want duplicate target error")
	}
	if !strings.Contains(err.Error(), "duplicate target ID") {
		t.Fatalf("run error = %q, want duplicate target context", err)
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

func TestRunValidatesSecretInputFlagForRequiredTargetSecret(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	catalogPath := writeLinuxSecretCatalog(t, "opensuse-secret-amd64-netboot")
	secretPath := filepath.Join(dir, "installer-password")
	if err := os.WriteFile(secretPath, []byte("secret value"), 0o600); err != nil {
		t.Fatalf("write secret: %v", err)
	}

	var stdout bytes.Buffer
	err := runWithIO(context.Background(), []string{
		"--mode", "plan-target",
		"--catalog", catalogPath,
		"--target", "opensuse-secret-amd64-netboot",
		"--secret", "installer-password=" + secretPath,
	}, strings.NewReader(""), &stdout, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if strings.Contains(stdout.String(), "secret value") || strings.Contains(stdout.String(), secretPath) {
		t.Fatalf("stdout exposed secret material: %q", stdout.String())
	}
}

func TestRunRejectsMissingRequiredSecret(t *testing.T) {
	t.Parallel()

	err := runWithIO(context.Background(), []string{
		"--mode", "plan-target",
		"--catalog", writeLinuxSecretCatalog(t, "opensuse-secret-amd64-netboot"),
		"--target", "opensuse-secret-amd64-netboot",
	}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if !errors.Is(err, provider.ErrInvalidSecretInput) {
		t.Fatalf("run error = %v, want invalid secret input", err)
	}
}

func TestRunDiagnosticsRedactsSecretInputPath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	catalogPath := writeLinuxSecretCatalog(t, "opensuse-secret-amd64-netboot")
	secretPath := filepath.Join(dir, "missing-secret")
	diagnosticsRoot := filepath.Join(dir, "diagnostics")
	err := runWithIO(context.Background(), []string{
		"--diagnostics-dir", diagnosticsRoot,
		"--mode", "plan-target",
		"--catalog", catalogPath,
		"--target", "opensuse-secret-amd64-netboot",
		"--secret", "installer-password=" + secretPath,
	}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("run succeeded, want missing secret error")
	}
	if !errors.Is(err, provider.ErrInvalidSecretInput) {
		t.Fatalf("run error = %v, want invalid secret input", err)
	}

	bundleDir := onlyDiagnosticsBundleDir(t, diagnosticsRoot)
	got := readDiagnosticsSummary(t, filepath.Join(bundleDir, "summary.json"))
	if !slices.Equal(got.SecretInputIDs, []string{"installer-password"}) {
		t.Fatalf("secret input IDs = %#v, want installer-password", got.SecretInputIDs)
	}
	if strings.Contains(got.Error, secretPath) {
		t.Fatalf("summary error = %q, want redacted secret path", got.Error)
	}
}

func TestSecretFlagsStringRedactsPaths(t *testing.T) {
	t.Parallel()

	var flags secretFlags
	if err := flags.Set("installer-password=/run/bootup/secrets/installer-password"); err != nil {
		t.Fatalf("set secret flag: %v", err)
	}
	got := flags.String()
	if strings.Contains(got, "/run/bootup/secrets/installer-password") {
		t.Fatalf("secret flag string = %q, want redacted path", got)
	}
	if !strings.Contains(got, "installer-password=<redacted>") {
		t.Fatalf("secret flag string = %q, want secret ID with redacted path", got)
	}
}

func writeLinuxCatalog(t *testing.T, targetID string) string {
	t.Helper()

	catalogPath := filepath.Join(t.TempDir(), "catalog.json")
	if err := os.WriteFile(catalogPath, []byte(`{
		"schema_version": 1,
		"targets": [{
			"id": "`+targetID+`",
			"provider_id": "linux",
			"name": "openSUSE lab amd64 netboot",
			"catalog": {
				"distribution": "opensuse",
				"release": "lab",
				"architecture": "amd64",
				"kind": "installer"
			},
			"source": {
				"base_url": "https://download.example/opensuse",
				"kernel_path": "boot/x86_64/loader/linux",
				"kernel_sha256": "`+strings.Repeat("a", 64)+`"
			}
		}]
	}`), 0o644); err != nil {
		t.Fatalf("write catalog: %v", err)
	}
	return catalogPath
}

func writeLinuxSecretCatalog(t *testing.T, targetID string) string {
	t.Helper()

	catalogPath := filepath.Join(t.TempDir(), "catalog.json")
	if err := os.WriteFile(catalogPath, []byte(`{
		"schema_version": 1,
		"targets": [{
			"id": "`+targetID+`",
			"provider_id": "linux",
			"name": "openSUSE secret amd64 netboot",
			"catalog": {
				"distribution": "opensuse",
				"release": "secret",
				"architecture": "amd64",
				"kind": "installer"
			},
			"source": {
				"base_url": "https://download.example/opensuse",
				"kernel_path": "boot/x86_64/loader/linux",
				"kernel_sha256": "`+strings.Repeat("a", 64)+`"
			},
			"secrets": [{
				"id": "installer-password",
				"label": "Installer password",
				"purpose": "automated installer login",
				"required": true,
				"delivery": "staged-file"
			}]
		}]
	}`), 0o644); err != nil {
		t.Fatalf("write catalog: %v", err)
	}
	return catalogPath
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

func TestRunPlansPolicySelectedTarget(t *testing.T) {
	t.Parallel()

	policyPath, signaturePath, publicKeyPath := writeSignedPolicyDecision(t, `{
		"schema_version": 1,
		"decision_id": "lab-opensuse",
		"target_id": "opensuse-leap-160-amd64-netboot",
		"options": {
			"text-install": "true",
			"mirror-url": "https://mirror.example/opensuse"
		},
		"expires_at": "2099-01-01T00:00:00Z"
	}`)

	var stdout bytes.Buffer
	err := runWithIO(context.Background(), []string{
		"--mode", "policy-target",
		"--policy-file", policyPath,
		"--policy-signature", signaturePath,
		"--policy-public-key", publicKeyPath,
	}, strings.NewReader(""), &stdout, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	for _, want := range []string{
		"[planning] openSUSE Leap 16.0 amd64 installer",
		"textmode=1",
		"install=https://mirror.example/opensuse",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
	if strings.Contains(stdout.String(), policyPath) || strings.Contains(stdout.String(), signaturePath) || strings.Contains(stdout.String(), publicKeyPath) {
		t.Fatalf("stdout exposed policy paths: %q", stdout.String())
	}
}

func TestRunPlansRemotePolicyFromAuthenticatedCacheFallback(t *testing.T) {
	t.Parallel()

	policyBytes, signaturePath, publicKeyPath := signedPolicyDecisionBytes(t, `{
		"schema_version": 1,
		"decision_id": "remote-lab-opensuse",
		"target_id": "opensuse-leap-160-amd64-netboot",
		"options": {
			"text-install": "true",
			"mirror-url": "https://mirror.example/opensuse"
		},
		"expires_at": "2099-01-01T00:00:00Z"
	}`)
	cachePath := filepath.Join(t.TempDir(), "policy-cache.json")
	if err := os.WriteFile(cachePath, policyBytes, 0o644); err != nil {
		t.Fatalf("write policy cache: %v", err)
	}

	var stdout bytes.Buffer
	err := runWithIO(context.Background(), []string{
		"--mode", "policy-target",
		"--policy-url", "https://127.0.0.1:1/policy.json",
		"--policy-signature", signaturePath,
		"--policy-public-key", publicKeyPath,
		"--policy-cache", cachePath,
		"--policy-cache-fallback",
		"--policy-timeout", "10ms",
	}, strings.NewReader(""), &stdout, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	for _, want := range []string{
		"[planning] openSUSE Leap 16.0 amd64 installer",
		"textmode=1",
		"install=https://mirror.example/opensuse",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
}

func TestRunRejectsLocalAndRemotePolicySources(t *testing.T) {
	t.Parallel()

	policyPath, signaturePath, publicKeyPath := writeSignedPolicyDecision(t, `{
		"schema_version": 1,
		"decision_id": "lab-opensuse",
		"target_id": "opensuse-leap-160-amd64-netboot",
		"expires_at": "2099-01-01T00:00:00Z"
	}`)

	err := runWithIO(context.Background(), []string{
		"--mode", "policy-target",
		"--policy-file", policyPath,
		"--policy-url", "https://policy.example/policy.json",
		"--policy-signature", signaturePath,
		"--policy-public-key", publicKeyPath,
	}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if !errors.Is(err, policy.ErrInvalidPolicy) {
		t.Fatalf("run error = %v, want invalid policy", err)
	}
	if !strings.Contains(err.Error(), "policy-url") {
		t.Fatalf("run error = %q, want policy-url context", err)
	}
}

func TestRunRejectsUnsupportedPolicyFallback(t *testing.T) {
	t.Parallel()

	err := runWithIO(context.Background(), []string{
		"--policy-fallback", "interactive",
	}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if !errors.Is(err, policy.ErrInvalidPolicy) {
		t.Fatalf("run error = %v, want invalid policy", err)
	}
	if !strings.Contains(err.Error(), "unsupported policy fallback") {
		t.Fatalf("run error = %q, want fallback context", err)
	}
}

func TestRunMenuPolicyFailureFallsBackToManualSelection(t *testing.T) {
	t.Parallel()

	policyPath, signaturePath, publicKeyPath := writeSignedPolicyDecision(t, `{
		"schema_version": 1,
		"decision_id": "expired-menu-opensuse",
		"target_id": "opensuse-leap-160-amd64-netboot",
		"expires_at": "2000-01-01T00:00:00Z"
	}`)

	var stdout bytes.Buffer
	err := runWithIO(context.Background(), []string{
		"--mode", "menu",
		"--ui", "plain",
		"--policy-file", policyPath,
		"--policy-signature", signaturePath,
		"--policy-public-key", publicKeyPath,
		"--policy-fallback", "manual",
	}, strings.NewReader("missing-target\n"), &stdout, &bytes.Buffer{})
	if err == nil {
		t.Fatal("run succeeded, want manual selection error")
	}
	if errors.Is(err, policy.ErrInvalidPolicy) {
		t.Fatalf("run error = %v, want manual selection error after fallback", err)
	}
	for _, want := range []string{
		"policy failure; falling back to manual target selection",
		"bootup targets",
		"boot option \"missing-target\" not found",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
}

func TestRunRejectsDynamicPolicyWithoutTrust(t *testing.T) {
	t.Parallel()

	policyPath, _, _ := writeSignedPolicyDecision(t, `{
		"schema_version": 1,
		"decision_id": "lab-opensuse",
		"target_id": "opensuse-leap-160-amd64-netboot",
		"expires_at": "2099-01-01T00:00:00Z"
	}`)

	err := runWithIO(context.Background(), []string{
		"--mode", "policy-target",
		"--policy-file", policyPath,
	}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if !errors.Is(err, policy.ErrInvalidPolicy) {
		t.Fatalf("run error = %v, want invalid policy", err)
	}
}

func TestRunDiagnosticsRedactsPolicyPaths(t *testing.T) {
	t.Parallel()

	policyPath, signaturePath, publicKeyPath := writeSignedPolicyDecision(t, `{
		"schema_version": 1,
		"decision_id": "expired-opensuse",
		"target_id": "opensuse-leap-160-amd64-netboot",
		"expires_at": "2000-01-01T00:00:00Z"
	}`)
	diagnosticsRoot := t.TempDir()

	err := runWithIO(context.Background(), []string{
		"--diagnostics-dir", diagnosticsRoot,
		"--mode", "policy-target",
		"--policy-file", policyPath,
		"--policy-signature", signaturePath,
		"--policy-public-key", publicKeyPath,
	}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if !errors.Is(err, policy.ErrInvalidPolicy) {
		t.Fatalf("run error = %v, want invalid policy", err)
	}

	bundleDir := onlyDiagnosticsBundleDir(t, diagnosticsRoot)
	summaryPath := filepath.Join(bundleDir, "summary.json")
	got := readDiagnosticsSummary(t, summaryPath)
	if got.Policy.Source != "local" || !got.Policy.Ed25519 || !got.Policy.Freshness {
		t.Fatalf("policy posture = %#v, want local Ed25519 freshness posture", got.Policy)
	}
	summaryBytes, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatalf("read summary: %v", err)
	}
	for _, secretPath := range []string{policyPath, signaturePath, publicKeyPath} {
		if strings.Contains(string(summaryBytes), secretPath) {
			t.Fatalf("summary exposes policy path %q: %s", secretPath, summaryBytes)
		}
	}
}

func TestRunDiagnosticsRedactsRemotePolicyURL(t *testing.T) {
	t.Parallel()

	policyBytes, signaturePath, publicKeyPath := signedPolicyDecisionBytes(t, `{
		"schema_version": 1,
		"decision_id": "expired-remote-opensuse",
		"target_id": "opensuse-leap-160-amd64-netboot",
		"expires_at": "2000-01-01T00:00:00Z"
	}`)
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "policy-cache.json")
	if err := os.WriteFile(cachePath, policyBytes, 0o644); err != nil {
		t.Fatalf("write policy cache: %v", err)
	}
	diagnosticsRoot := filepath.Join(dir, "diagnostics")
	secretURL := "https://127.0.0.1:1/policy.json?token=secret-token"

	err := runWithIO(context.Background(), []string{
		"--diagnostics-dir", diagnosticsRoot,
		"--mode", "policy-target",
		"--policy-url", secretURL,
		"--policy-signature", signaturePath,
		"--policy-public-key", publicKeyPath,
		"--policy-cache", cachePath,
		"--policy-cache-fallback",
		"--policy-timeout", "10ms",
	}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if !errors.Is(err, policy.ErrInvalidPolicy) {
		t.Fatalf("run error = %v, want invalid policy", err)
	}

	bundleDir := onlyDiagnosticsBundleDir(t, diagnosticsRoot)
	summaryPath := filepath.Join(bundleDir, "summary.json")
	got := readDiagnosticsSummary(t, summaryPath)
	if got.Policy.Source != "remote" || !got.Policy.RemoteURLSet || !got.Policy.CacheFallback {
		t.Fatalf("policy posture = %#v, want remote cache fallback posture", got.Policy)
	}
	summaryBytes, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatalf("read summary: %v", err)
	}
	for _, secret := range []string{secretURL, "secret-token", signaturePath, publicKeyPath, cachePath} {
		if strings.Contains(string(summaryBytes), secret) {
			t.Fatalf("summary exposes policy secret %q: %s", secret, summaryBytes)
		}
	}
}

func writeSignedPolicyDecision(t *testing.T, body string) (string, string, string) {
	t.Helper()

	data, signaturePath, publicKeyPath := signedPolicyDecisionBytes(t, body)
	policyPath := filepath.Join(t.TempDir(), "policy.json")
	if err := os.WriteFile(policyPath, data, 0o644); err != nil {
		t.Fatalf("write policy: %v", err)
	}
	return policyPath, signaturePath, publicKeyPath
}

func signedPolicyDecisionBytes(t *testing.T, body string) ([]byte, string, string) {
	t.Helper()

	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	dir := t.TempDir()
	data := []byte(body)
	signaturePath := filepath.Join(dir, "policy.json.sig")
	if err := os.WriteFile(signaturePath, ed25519.Sign(privateKey, data), 0o644); err != nil {
		t.Fatalf("write signature: %v", err)
	}
	publicKeyPath := filepath.Join(dir, "policy.pub")
	if err := os.WriteFile(publicKeyPath, publicKey, 0o644); err != nil {
		t.Fatalf("write public key: %v", err)
	}
	return data, signaturePath, publicKeyPath
}

func TestRunWritesDiagnosticsOnFailure(t *testing.T) {
	t.Parallel()

	diagnosticsRoot := t.TempDir()
	secretMirror := "https://secret.example/install token"
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runWithIO(context.Background(), []string{
		"--diagnostics-dir", diagnosticsRoot,
		"--mode", "plan-target",
		"--target", "opensuse-leap-160-amd64-netboot",
		"--option", "mirror-url=" + secretMirror,
	}, strings.NewReader(""), &stdout, &stderr)
	if err == nil {
		t.Fatal("run succeeded, want invalid option error")
	}
	if !errors.Is(err, provider.ErrInvalidTargetOption) {
		t.Fatalf("run error = %v, want invalid target option", err)
	}

	bundleDir := onlyDiagnosticsBundleDir(t, diagnosticsRoot)
	got := readDiagnosticsSummary(t, filepath.Join(bundleDir, "summary.json"))
	if got.Mode != "plan-target" {
		t.Fatalf("mode = %q, want plan-target", got.Mode)
	}
	if got.TargetID != "opensuse-leap-160-amd64-netboot" {
		t.Fatalf("target ID = %q, want selected target", got.TargetID)
	}
	if !slices.Equal(got.SelectedOptionIDs, []string{"mirror-url"}) {
		t.Fatalf("selected option IDs = %#v, want mirror-url", got.SelectedOptionIDs)
	}
	if got.Catalog.Source != "embedded" {
		t.Fatalf("catalog posture = %#v, want embedded source", got.Catalog)
	}
	if got.ProviderConfig.PathSet {
		t.Fatalf("provider config posture = %#v, want no path", got.ProviderConfig)
	}
	if strings.Contains(got.Error, secretMirror) || strings.Contains(got.Error, "token") {
		t.Fatalf("summary error = %q, want redacted selected option value", got.Error)
	}
	if !strings.Contains(got.Error, "<redacted>") {
		t.Fatalf("summary error = %q, want redaction marker", got.Error)
	}
	if got.CreatedAt == "" {
		t.Fatal("created_at is empty")
	}
	if got.SchemaVersion != 1 {
		t.Fatalf("schema version = %d, want 1", got.SchemaVersion)
	}

	stdoutFile := readTextFile(t, filepath.Join(bundleDir, "stdout.txt"))
	if stdoutFile != stdout.String() {
		t.Fatalf("captured stdout = %q, want console stdout %q", stdoutFile, stdout.String())
	}
	if !strings.Contains(stdoutFile, "[planning] openSUSE Leap 16.0 amd64 installer") {
		t.Fatalf("captured stdout = %q, want planning output", stdoutFile)
	}
	stderrFile := readTextFile(t, filepath.Join(bundleDir, "stderr.txt"))
	if stderrFile != stderr.String() {
		t.Fatalf("captured stderr = %q, want console stderr %q", stderrFile, stderr.String())
	}
	if !strings.Contains(stderrFile, "bootup started") {
		t.Fatalf("captured stderr = %q, want app log output", stderrFile)
	}
}

func TestRunDiagnosticsDisabledByDefault(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runWithIO(context.Background(), []string{
		"--mode", "plan-target",
		"--target", "opensuse-leap-160-amd64-netboot",
		"--option", "mirror-url=https://secret.example/install token",
	}, strings.NewReader(""), &stdout, &stderr)
	if err == nil {
		t.Fatal("run succeeded, want invalid option error")
	}
	if strings.Contains(err.Error(), "diagnostics") {
		t.Fatalf("run error = %q, want no diagnostics context by default", err)
	}
	if strings.Contains(stdout.String(), "summary.json") || strings.Contains(stderr.String(), "summary.json") {
		t.Fatalf("output mentions diagnostics by default: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestRunMirrorsOutputToConsolePath(t *testing.T) {
	t.Parallel()

	consolePath := filepath.Join(t.TempDir(), "tty0")
	if err := os.WriteFile(consolePath, nil, 0o600); err != nil {
		t.Fatalf("create console mirror: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runWithIO(context.Background(), []string{
		"--mode", "list-targets",
		"--console-mirror", consolePath,
	}, strings.NewReader(""), &stdout, &stderr)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	mirror := readTextFile(t, consolePath)
	for _, want := range []string{
		"bootup: console mirror enabled path=" + consolePath,
		"bootup targets",
		"bootup started",
	} {
		if !strings.Contains(mirror, want) {
			t.Fatalf("mirror output = %q, want %q", mirror, want)
		}
	}
	if !strings.Contains(stdout.String(), "bootup targets") {
		t.Fatalf("stdout = %q, want normal target output", stdout.String())
	}
	if !strings.Contains(stderr.String(), "console mirror enabled") {
		t.Fatalf("stderr = %q, want console mirror notice", stderr.String())
	}
}

func TestRunSkipsConsoleMirrorWhenPathAlreadyReceivesOutput(t *testing.T) {
	t.Parallel()

	consolePath := filepath.Join(t.TempDir(), "console")
	output, err := os.OpenFile(consolePath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatalf("open console output: %v", err)
	}
	t.Cleanup(func() { _ = output.Close() })

	err = runWithIO(context.Background(), []string{
		"--mode", "list-targets",
		"--console-mirror", consolePath,
	}, strings.NewReader(""), output, output)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	got := readTextFile(t, consolePath)
	for _, want := range []string{"bootup started", "bootup targets"} {
		if count := strings.Count(got, want); count != 1 {
			t.Fatalf("output contains %q %d times, want 1: %q", want, count, got)
		}
	}
	if strings.Contains(got, "console mirror enabled") {
		t.Fatalf("output = %q, did not want mirror enabled notice", got)
	}
}

func TestRunReportsDiagnosticsWriteFailureAsSecondary(t *testing.T) {
	t.Parallel()

	rootFile := filepath.Join(t.TempDir(), "diagnostics")
	if err := os.WriteFile(rootFile, []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("write root file: %v", err)
	}

	err := runWithIO(context.Background(), []string{
		"--diagnostics-dir", rootFile,
		"--mode", "plan-target",
		"--target", "opensuse-leap-160-amd64-netboot",
		"--option", "mirror-url=https://secret.example/install token",
	}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("run succeeded, want invalid option error")
	}
	if !errors.Is(err, provider.ErrInvalidTargetOption) {
		t.Fatalf("run error = %v, want invalid target option", err)
	}
	if !strings.Contains(err.Error(), "write diagnostics") {
		t.Fatalf("run error = %q, want secondary diagnostics write context", err)
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

func onlyDiagnosticsBundleDir(t *testing.T, root string) string {
	t.Helper()

	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read diagnostics root: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("diagnostics entries = %d, want 1", len(entries))
	}
	if !entries[0].IsDir() {
		t.Fatalf("diagnostics entry %q is not a directory", entries[0].Name())
	}
	return filepath.Join(root, entries[0].Name())
}

func readDiagnosticsSummary(t *testing.T, path string) diagnostics.Summary {
	t.Helper()

	data := []byte(readTextFile(t, path))
	var summary diagnostics.Summary
	if err := json.Unmarshal(data, &summary); err != nil {
		t.Fatalf("decode diagnostics summary: %v", err)
	}
	return summary
}

func readTextFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
