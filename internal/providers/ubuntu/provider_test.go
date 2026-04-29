package ubuntu_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/dotwaffle/bootup/internal/provider"
	"github.com/dotwaffle/bootup/internal/providers/ubuntu"
)

func TestProviderTargetsUbuntu2604AMD64(t *testing.T) {
	t.Parallel()

	targets, err := ubuntu.NewProvider(ubuntu.Config{}).Targets(context.Background())
	if err != nil {
		t.Fatalf("targets: %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("targets length = %d, want 1", len(targets))
	}
	target := targets[0]
	if target.ID != "ubuntu-2604-amd64-netboot" {
		t.Fatalf("target ID = %q, want Ubuntu 26.04", target.ID)
	}
	if target.ProviderID != "ubuntu" {
		t.Fatalf("provider ID = %q, want ubuntu", target.ProviderID)
	}
	if target.Architecture != "amd64" {
		t.Fatalf("architecture = %q, want amd64", target.Architecture)
	}
	if target.Distribution != "ubuntu" {
		t.Fatalf("distribution = %q, want ubuntu", target.Distribution)
	}
	if target.Release != "26.04" {
		t.Fatalf("release = %q, want 26.04", target.Release)
	}
	if target.Kind != "installer" {
		t.Fatalf("kind = %q, want installer", target.Kind)
	}
}

func TestProviderPlanResolvesReleaseURLs(t *testing.T) {
	t.Parallel()

	p := ubuntu.NewProvider(ubuntu.Config{
		ReleaseURL:   "https://mirror.example/26.04",
		KernelSHA256: strings.Repeat("a", 64),
		InitrdSHA256: strings.Repeat("b", 64),
	})
	plan, err := p.Plan(context.Background(), provider.Target{
		ID:         "ubuntu-2604-amd64-netboot",
		ProviderID: "ubuntu",
	})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}

	if plan.Kernel.URL != "https://mirror.example/26.04/netboot/amd64/linux" {
		t.Fatalf("kernel URL = %q", plan.Kernel.URL)
	}
	if plan.Initrd.URL != "https://mirror.example/26.04/netboot/amd64/initrd" {
		t.Fatalf("initrd URL = %q", plan.Initrd.URL)
	}
	if plan.Verification.ChecksumURL != "https://mirror.example/26.04/SHA256SUMS" {
		t.Fatalf("checksum URL = %q", plan.Verification.ChecksumURL)
	}
	if plan.Verification.SignatureURL != "https://mirror.example/26.04/SHA256SUMS.gpg" {
		t.Fatalf("signature URL = %q", plan.Verification.SignatureURL)
	}
	if !strings.Contains(plan.Cmdline, "url=https://mirror.example/26.04/ubuntu-26.04-live-server-amd64.iso") {
		t.Fatalf("cmdline = %q, want ISO URL", plan.Cmdline)
	}
	if plan.Kernel.SHA256 != strings.Repeat("a", 64) {
		t.Fatalf("kernel sha256 = %q", plan.Kernel.SHA256)
	}
}

func TestFetchAndStageArtifactsAllowsHTTPSOnlyNetboot(t *testing.T) {
	t.Parallel()

	kernel := []byte("kernel")
	initrd := []byte("initrd")
	plan := provider.BootPlan{
		Target: provider.Target{ID: "ubuntu-2604-amd64-netboot", ProviderID: "ubuntu"},
		Kernel: provider.Artifact{Name: "linux", URL: "https://mirror.example/26.04/netboot/amd64/linux"},
		Initrd: provider.Artifact{Name: "initrd", URL: "https://mirror.example/26.04/netboot/amd64/initrd"},
	}
	client := &http.Client{Transport: responseMap{
		"https://mirror.example/26.04/netboot/amd64/linux":  kernel,
		"https://mirror.example/26.04/netboot/amd64/initrd": initrd,
	}}

	staged, err := ubuntu.FetchAndStageArtifacts(context.Background(), ubuntu.FetchConfig{
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
}

func TestFetchAndStageArtifactsRejectsUnverifiedHTTP(t *testing.T) {
	t.Parallel()

	plan := provider.BootPlan{
		Target: provider.Target{ID: "ubuntu-2604-amd64-netboot", ProviderID: "ubuntu"},
		Kernel: provider.Artifact{Name: "linux", URL: "http://mirror.example/26.04/netboot/amd64/linux"},
		Initrd: provider.Artifact{Name: "initrd", URL: "http://mirror.example/26.04/netboot/amd64/initrd"},
	}

	_, err := ubuntu.FetchAndStageArtifacts(context.Background(), ubuntu.FetchConfig{
		Plan:       plan,
		StagingDir: t.TempDir(),
	})
	if err == nil {
		t.Fatal("fetch and stage succeeded, want https failure")
	}
}

func TestFetchAndStageArtifactsVerifiesSignedMetadataAndArtifacts(t *testing.T) {
	t.Parallel()

	kernel := []byte("kernel")
	initrd := []byte("initrd")
	iso := []byte("iso")
	kernelSum := sha256.Sum256(kernel)
	initrdSum := sha256.Sum256(initrd)
	isoSum := sha256.Sum256(iso)
	shaSums := fmt.Appendf(nil, "%x *ubuntu-26.04-live-server-amd64.iso\n", isoSum)
	keyring, signature := signedSHA256Sums(t, shaSums)

	client := &http.Client{Transport: responseMap{
		"https://mirror.example/26.04/SHA256SUMS":           shaSums,
		"https://mirror.example/26.04/SHA256SUMS.gpg":       signature,
		"https://mirror.example/26.04/netboot/amd64/linux":  kernel,
		"https://mirror.example/26.04/netboot/amd64/initrd": initrd,
	}}

	p := ubuntu.NewProvider(ubuntu.Config{
		ReleaseURL:   "https://mirror.example/26.04",
		KernelSHA256: hex.EncodeToString(kernelSum[:]),
		InitrdSHA256: hex.EncodeToString(initrdSum[:]),
	})
	plan, err := p.Plan(context.Background(), provider.Target{
		ID:         "ubuntu-2604-amd64-netboot",
		ProviderID: "ubuntu",
	})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}

	staged, err := ubuntu.FetchAndStageArtifacts(context.Background(), ubuntu.FetchConfig{
		Plan:       plan,
		Client:     client,
		Keyring:    bytes.NewReader(keyring),
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
	if filepath.Base(staged.Initrd.Path) != "initrd" {
		t.Fatalf("staged initrd path = %q, want initrd", staged.Initrd.Path)
	}
}

func TestProviderStageUsesConfiguredTrust(t *testing.T) {
	t.Parallel()

	kernel := []byte("kernel")
	initrd := []byte("initrd")
	isoSum := sha256.Sum256([]byte("iso"))
	kernelSum := sha256.Sum256(kernel)
	initrdSum := sha256.Sum256(initrd)
	shaSums := fmt.Appendf(nil, "%x *ubuntu-26.04-live-server-amd64.iso\n", isoSum)
	keyring, signature := signedSHA256Sums(t, shaSums)

	client := &http.Client{Transport: responseMap{
		"https://mirror.example/26.04/SHA256SUMS":           shaSums,
		"https://mirror.example/26.04/SHA256SUMS.gpg":       signature,
		"https://mirror.example/26.04/netboot/amd64/linux":  kernel,
		"https://mirror.example/26.04/netboot/amd64/initrd": initrd,
	}}
	p := ubuntu.NewProvider(ubuntu.Config{
		ReleaseURL:   "https://mirror.example/26.04",
		Client:       client,
		Keyring:      keyring,
		KernelSHA256: hex.EncodeToString(kernelSum[:]),
		InitrdSHA256: hex.EncodeToString(initrdSum[:]),
	})
	plan, err := p.Plan(context.Background(), provider.Target{
		ID:         "ubuntu-2604-amd64-netboot",
		ProviderID: "ubuntu",
	})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}

	staged, err := p.Stage(context.Background(), provider.StageConfig{
		Plan:       plan,
		StagingDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("stage: %v", err)
	}
	if filepath.Base(staged.Kernel.Path) != "linux" {
		t.Fatalf("staged kernel path = %q, want linux", staged.Kernel.Path)
	}
}

func signedSHA256Sums(t *testing.T, sums []byte) ([]byte, []byte) {
	t.Helper()

	entity, err := openpgp.NewEntity("Ubuntu Release", "", "ubuntu@example.test", nil)
	if err != nil {
		t.Fatalf("new entity: %v", err)
	}

	var signature bytes.Buffer
	if err := openpgp.DetachSign(&signature, entity, bytes.NewReader(sums), nil); err != nil {
		t.Fatalf("detach sign: %v", err)
	}

	var keyring bytes.Buffer
	armorWriter, err := armor.Encode(&keyring, openpgp.PublicKeyType, nil)
	if err != nil {
		t.Fatalf("armor keyring: %v", err)
	}
	if err := entity.Serialize(armorWriter); err != nil {
		t.Fatalf("serialize keyring: %v", err)
	}
	if err := armorWriter.Close(); err != nil {
		t.Fatalf("close armor: %v", err)
	}

	return keyring.Bytes(), signature.Bytes()
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
