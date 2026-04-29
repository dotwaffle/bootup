package debian_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/clearsign"
	"github.com/dotwaffle/bootup/internal/provider"
	"github.com/dotwaffle/bootup/internal/providers/debian"
)

func TestProviderTargetsDebianTrixieAMD64(t *testing.T) {
	t.Parallel()

	targets, err := debian.NewProvider(debian.Config{}).Targets(context.Background())
	if err != nil {
		t.Fatalf("targets: %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("targets length = %d, want 1", len(targets))
	}
	target := targets[0]
	if target.ID != "debian-trixie-amd64-netboot" {
		t.Fatalf("target ID = %q, want Debian trixie", target.ID)
	}
	if target.ProviderID != "debian" {
		t.Fatalf("provider ID = %q, want debian", target.ProviderID)
	}
	if target.Architecture != "amd64" {
		t.Fatalf("architecture = %q, want amd64", target.Architecture)
	}
}

func TestProviderPlanResolvesInstallerURLs(t *testing.T) {
	t.Parallel()

	p := debian.NewProvider(debian.Config{MirrorURL: "https://mirror.example/debian"})
	plan, err := p.Plan(context.Background(), provider.Target{
		ID:         "debian-trixie-amd64-netboot",
		ProviderID: "debian",
	})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}

	if plan.Kernel.URL != "https://mirror.example/debian/dists/trixie/main/installer-amd64/current/images/netboot/debian-installer/amd64/linux" {
		t.Fatalf("kernel URL = %q", plan.Kernel.URL)
	}
	if plan.Initrd.URL != "https://mirror.example/debian/dists/trixie/main/installer-amd64/current/images/netboot/debian-installer/amd64/initrd.gz" {
		t.Fatalf("initrd URL = %q", plan.Initrd.URL)
	}
	if plan.Verification.MetadataURL != "https://mirror.example/debian/dists/trixie/InRelease" {
		t.Fatalf("metadata URL = %q", plan.Verification.MetadataURL)
	}
	if plan.Verification.ChecksumURL != "https://mirror.example/debian/dists/trixie/main/installer-amd64/current/images/SHA256SUMS" {
		t.Fatalf("checksum URL = %q", plan.Verification.ChecksumURL)
	}
	if plan.Cmdline == "" {
		t.Fatal("cmdline is empty")
	}
}

func TestVerifyInReleaseReturnsTrustedPlaintext(t *testing.T) {
	t.Parallel()

	keyring, signed := signedRelease(t, []byte("SHA256:\n abc123 1 path/to/file\n"))
	plaintext, err := debian.VerifyInRelease(signed, bytes.NewReader(keyring))
	if err != nil {
		t.Fatalf("verify InRelease: %v", err)
	}
	if !strings.Contains(string(plaintext), "SHA256:") {
		t.Fatalf("plaintext = %q, want SHA256 stanza", plaintext)
	}
}

func TestVerifyInReleaseRejectsUntrustedSignature(t *testing.T) {
	t.Parallel()

	_, signed := signedRelease(t, []byte("SHA256:\n abc123 1 path/to/file\n"))
	untrustedKeyring, _ := signedRelease(t, []byte("different signer"))

	_, err := debian.VerifyInRelease(signed, bytes.NewReader(untrustedKeyring))
	if err == nil {
		t.Fatal("verify InRelease succeeded, want untrusted signature failure")
	}
}

func TestVerifyArtifactChecksumAcceptsTrustedHash(t *testing.T) {
	t.Parallel()

	data := []byte("kernel")
	sum := sha256.Sum256(data)
	checksumsLine := fmt.Appendf(nil, "%x  debian-installer/amd64/linux\n", sum)
	checksums, err := debian.ParseSHA256Sums(checksumsLine)
	if err != nil {
		t.Fatalf("parse checksums: %v", err)
	}

	if err := debian.VerifyArtifactChecksum("debian-installer/amd64/linux", data, checksums); err != nil {
		t.Fatalf("verify checksum: %v", err)
	}
}

func TestVerifyArtifactChecksumRejectsMismatch(t *testing.T) {
	t.Parallel()

	checksums, err := debian.ParseSHA256Sums([]byte(strings.Repeat("0", 64) + "  debian-installer/amd64/linux\n"))
	if err != nil {
		t.Fatalf("parse checksums: %v", err)
	}

	err = debian.VerifyArtifactChecksum("debian-installer/amd64/linux", []byte("kernel"), checksums)
	if err == nil {
		t.Fatal("verify checksum succeeded, want failure")
	}
}

func TestFetchAndStageArtifactsVerifiesSignedMetadataAndArtifacts(t *testing.T) {
	t.Parallel()

	kernel := []byte("kernel")
	initrd := []byte("initrd")
	kernelSum := sha256.Sum256(kernel)
	initrdSum := sha256.Sum256(initrd)
	shaSums := fmt.Appendf(nil,
		"%x  ./netboot/debian-installer/amd64/linux\n%x  ./netboot/debian-installer/amd64/initrd.gz\n",
		kernelSum,
		initrdSum,
	)
	shaSumsSum := sha256.Sum256(shaSums)

	release := fmt.Appendf(nil, "SHA256:\n %x %d main/installer-amd64/current/images/SHA256SUMS\n", shaSumsSum, len(shaSums))
	keyring, signed := signedRelease(t, release)

	client := &http.Client{Transport: responseMap{
		"https://mirror.example/debian/dists/trixie/InRelease":                                                                    signed,
		"https://mirror.example/debian/dists/trixie/main/installer-amd64/current/images/SHA256SUMS":                               shaSums,
		"https://mirror.example/debian/dists/trixie/main/installer-amd64/current/images/netboot/debian-installer/amd64/linux":     kernel,
		"https://mirror.example/debian/dists/trixie/main/installer-amd64/current/images/netboot/debian-installer/amd64/initrd.gz": initrd,
	}}

	p := debian.NewProvider(debian.Config{MirrorURL: "https://mirror.example/debian"})
	plan, err := p.Plan(context.Background(), provider.Target{
		ID:         "debian-trixie-amd64-netboot",
		ProviderID: "debian",
	})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}

	staged, err := debian.FetchAndStageArtifacts(context.Background(), debian.FetchConfig{
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
	if filepath.Base(staged.Initrd.Path) != "initrd.gz" {
		t.Fatalf("staged initrd path = %q, want initrd.gz", staged.Initrd.Path)
	}
}

func TestProviderStageUsesConfiguredKeyring(t *testing.T) {
	t.Parallel()

	kernel := []byte("kernel")
	initrd := []byte("initrd")
	kernelSum := sha256.Sum256(kernel)
	initrdSum := sha256.Sum256(initrd)
	shaSums := fmt.Appendf(nil,
		"%x  ./netboot/debian-installer/amd64/linux\n%x  ./netboot/debian-installer/amd64/initrd.gz\n",
		kernelSum,
		initrdSum,
	)
	shaSumsSum := sha256.Sum256(shaSums)

	release := fmt.Appendf(nil, "SHA256:\n %x %d main/installer-amd64/current/images/SHA256SUMS\n", shaSumsSum, len(shaSums))
	keyring, signed := signedRelease(t, release)

	client := &http.Client{Transport: responseMap{
		"https://mirror.example/debian/dists/trixie/InRelease":                                                                    signed,
		"https://mirror.example/debian/dists/trixie/main/installer-amd64/current/images/SHA256SUMS":                               shaSums,
		"https://mirror.example/debian/dists/trixie/main/installer-amd64/current/images/netboot/debian-installer/amd64/linux":     kernel,
		"https://mirror.example/debian/dists/trixie/main/installer-amd64/current/images/netboot/debian-installer/amd64/initrd.gz": initrd,
	}}

	p := debian.NewProvider(debian.Config{
		MirrorURL: "https://mirror.example/debian",
		Client:    client,
		Keyring:   keyring,
	})
	target := provider.Target{
		ID:         "debian-trixie-amd64-netboot",
		ProviderID: "debian",
	}
	plan, err := p.Plan(context.Background(), target)
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
	if filepath.Base(staged.Initrd.Path) != "initrd.gz" {
		t.Fatalf("staged initrd path = %q, want initrd.gz", staged.Initrd.Path)
	}
}

func signedRelease(t *testing.T, release []byte) ([]byte, []byte) {
	t.Helper()

	entity, err := openpgp.NewEntity("Debian Archive", "", "debian@example.test", nil)
	if err != nil {
		t.Fatalf("new entity: %v", err)
	}

	var signed bytes.Buffer
	plaintext, err := clearsign.Encode(&signed, entity.PrivateKey, nil)
	if err != nil {
		t.Fatalf("clearsign encode: %v", err)
	}
	if _, err := plaintext.Write(release); err != nil {
		t.Fatalf("write release: %v", err)
	}
	if err := plaintext.Close(); err != nil {
		t.Fatalf("close clearsign: %v", err)
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

	return keyring.Bytes(), signed.Bytes()
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
