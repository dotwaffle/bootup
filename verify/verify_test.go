package verify_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/dotwaffle/bootup/verify"
)

func TestSHA256AcceptsExpectedHash(t *testing.T) {
	t.Parallel()

	sum := sha256.Sum256([]byte("kernel"))

	if err := verify.SHA256(verify.HashInput{
		Artifact:       bytes.NewReader([]byte("kernel")),
		ExpectedSHA256: hex.EncodeToString(sum[:]),
		Name:           "linux",
	}); err != nil {
		t.Fatalf("verify sha256: %v", err)
	}
}

func TestArtifactRunsAllSuppliedChecks(t *testing.T) {
	t.Parallel()

	data := []byte("kernel")
	sum := sha256.Sum256(data)
	sums := []byte(hex.EncodeToString(sum[:]) + "  debian-installer/amd64/linux\n")
	entity := testEntity(t)

	if err := verify.Artifact(verify.ArtifactInput{
		Artifact:       bytes.NewReader(data),
		Name:           "debian-installer/amd64/linux",
		ExpectedSHA256: hex.EncodeToString(sum[:]),
		SHA256Sums:     bytes.NewReader(sums),
		Signature:      bytes.NewReader(detachedSignature(t, entity, data)),
		Keyring:        bytes.NewReader(publicKeyring(t, entity)),
	}); err != nil {
		t.Fatalf("verify artifact: %v", err)
	}
}

func TestArtifactRequiresVerificationMaterial(t *testing.T) {
	t.Parallel()

	err := verify.Artifact(verify.ArtifactInput{
		Artifact: bytes.NewReader([]byte("kernel")),
		Name:     "linux",
	})
	if err == nil {
		t.Fatal("verify artifact succeeded, want missing verification material failure")
	}
}

func TestArtifactRequiresAllSuppliedChecksToPass(t *testing.T) {
	t.Parallel()

	data := []byte("kernel")
	entity := testEntity(t)

	err := verify.Artifact(verify.ArtifactInput{
		Artifact:       bytes.NewReader(data),
		Name:           "linux",
		ExpectedSHA256: strings.Repeat("0", 64),
		Signature:      bytes.NewReader(detachedSignature(t, entity, data)),
		Keyring:        bytes.NewReader(publicKeyring(t, entity)),
	})
	if err == nil {
		t.Fatal("verify artifact succeeded, want hash failure")
	}
}

func TestArtifactSignatureRequiresKeyring(t *testing.T) {
	t.Parallel()

	entity := testEntity(t)
	err := verify.Artifact(verify.ArtifactInput{
		Artifact:  bytes.NewReader([]byte("kernel")),
		Name:      "linux",
		Signature: bytes.NewReader(detachedSignature(t, entity, []byte("kernel"))),
	})
	if err == nil {
		t.Fatal("verify artifact succeeded, want missing keyring failure")
	}
}

func TestSHA256RejectsMismatchedHash(t *testing.T) {
	t.Parallel()

	err := verify.SHA256(verify.HashInput{
		Artifact:       bytes.NewReader([]byte("kernel")),
		ExpectedSHA256: strings.Repeat("0", 64),
		Name:           "linux",
	})
	if err == nil {
		t.Fatal("verify sha256 succeeded, want mismatch")
	}
}

func TestSHA256SumsAcceptsExpectedHash(t *testing.T) {
	t.Parallel()

	data := []byte("kernel")
	sum := sha256.Sum256(data)
	sums := []byte(hex.EncodeToString(sum[:]) + "  debian-installer/amd64/linux\n")

	if err := verify.SHA256Sums(verify.SumFileInput{
		Artifact: bytes.NewReader(data),
		Sums:     bytes.NewReader(sums),
		Name:     "debian-installer/amd64/linux",
	}); err != nil {
		t.Fatalf("verify sha256 sums: %v", err)
	}
}

func TestSHA256SumsNormalizesDebianRelativePath(t *testing.T) {
	t.Parallel()

	data := []byte("kernel")
	sum := sha256.Sum256(data)
	sums := []byte(hex.EncodeToString(sum[:]) + "  ./netboot/debian-installer/amd64/linux\n")

	if err := verify.SHA256Sums(verify.SumFileInput{
		Artifact: bytes.NewReader(data),
		Sums:     bytes.NewReader(sums),
		Name:     "netboot/debian-installer/amd64/linux",
	}); err != nil {
		t.Fatalf("verify sha256 sums: %v", err)
	}
}

func TestSHA256FileAcceptsExpectedHash(t *testing.T) {
	t.Parallel()

	path := writeFile(t, "artifact", []byte("kernel"))
	sum := sha256.Sum256([]byte("kernel"))

	if err := verify.SHA256File(verify.FileHashInput{
		Path:           path,
		ExpectedSHA256: hex.EncodeToString(sum[:]),
	}); err != nil {
		t.Fatalf("verify sha256 file: %v", err)
	}
}

func TestArmoredDetachedSignatureAcceptsTrustedSignature(t *testing.T) {
	t.Parallel()

	entity := testEntity(t)
	signature := detachedSignature(t, entity, []byte("payload"))
	keyring := publicKeyring(t, entity)

	if err := verify.ArmoredDetachedSignature(verify.SignatureInput{
		Artifact:  bytes.NewReader([]byte("payload")),
		Signature: bytes.NewReader(signature),
		Keyring:   bytes.NewReader(keyring),
		Name:      "artifact",
	}); err != nil {
		t.Fatalf("verify detached signature: %v", err)
	}
}

func TestArmoredDetachedSignatureRejectsUntrustedSignature(t *testing.T) {
	t.Parallel()

	signature := detachedSignature(t, testEntity(t), []byte("payload"))
	keyring := publicKeyring(t, testEntity(t))

	err := verify.ArmoredDetachedSignature(verify.SignatureInput{
		Artifact:  bytes.NewReader([]byte("payload")),
		Signature: bytes.NewReader(signature),
		Keyring:   bytes.NewReader(keyring),
		Name:      "artifact",
	})
	if err == nil {
		t.Fatal("verify detached signature succeeded, want untrusted signature failure")
	}
}

func TestArmoredDetachedSignatureFileAcceptsTrustedSignature(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	artifact := filepath.Join(dir, "artifact")
	signature := filepath.Join(dir, "artifact.asc")
	keyring := filepath.Join(dir, "archive-keyring.asc")
	entity := testEntity(t)

	if err := os.WriteFile(artifact, []byte("payload"), 0o644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	if err := os.WriteFile(signature, detachedSignature(t, entity, []byte("payload")), 0o644); err != nil {
		t.Fatalf("write signature: %v", err)
	}
	if err := os.WriteFile(keyring, publicKeyring(t, entity), 0o644); err != nil {
		t.Fatalf("write keyring: %v", err)
	}

	if err := verify.ArmoredDetachedSignatureFile(verify.FileSignatureInput{
		ArtifactPath:  artifact,
		SignaturePath: signature,
		KeyringPath:   keyring,
	}); err != nil {
		t.Fatalf("verify detached signature file: %v", err)
	}
}

func TestReadKeyringAcceptsBinaryOpenPGPKeyring(t *testing.T) {
	t.Parallel()

	if _, err := verify.ReadKeyring(bytes.NewReader(binaryKeyring(t, testEntity(t)))); err != nil {
		t.Fatalf("read binary keyring: %v", err)
	}
}

func writeFile(t *testing.T, name string, data []byte) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	return path
}

func testEntity(t *testing.T) *openpgp.Entity {
	t.Helper()

	entity, err := openpgp.NewEntity("Archive", "", "archive@example.test", nil)
	if err != nil {
		t.Fatalf("new entity: %v", err)
	}
	return entity
}

func detachedSignature(t *testing.T, entity *openpgp.Entity, data []byte) []byte {
	t.Helper()

	var signature bytes.Buffer
	if err := openpgp.ArmoredDetachSign(&signature, entity, bytes.NewReader(data), nil); err != nil {
		t.Fatalf("detach sign: %v", err)
	}
	return signature.Bytes()
}

func publicKeyring(t *testing.T, entity *openpgp.Entity) []byte {
	t.Helper()

	var keyring bytes.Buffer
	writer, err := armor.Encode(&keyring, openpgp.PublicKeyType, nil)
	if err != nil {
		t.Fatalf("armor keyring: %v", err)
	}
	if err := entity.Serialize(writer); err != nil {
		t.Fatalf("serialize keyring: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close keyring: %v", err)
	}
	return keyring.Bytes()
}

func binaryKeyring(t *testing.T, entity *openpgp.Entity) []byte {
	t.Helper()

	var keyring bytes.Buffer
	if err := entity.Serialize(&keyring); err != nil {
		t.Fatalf("serialize binary keyring: %v", err)
	}
	return keyring.Bytes()
}
