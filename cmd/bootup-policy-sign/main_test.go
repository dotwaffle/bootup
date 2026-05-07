package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunGeneratesKeyPairAndSignsPolicy(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	privateKeyPath := filepath.Join(dir, "policy.key")
	publicKeyPath := filepath.Join(dir, "policy.pub")
	policyPath := filepath.Join(dir, "policy.json")
	signaturePath := filepath.Join(dir, "policy.json.sig")
	policyBytes := []byte(`{"schema_version":1,"decision_id":"site-a","target_id":"opensuse-leap-160-amd64-netboot","expires_at":"2099-01-01T00:00:00Z"}`)
	if err := os.WriteFile(policyPath, policyBytes, 0o644); err != nil {
		t.Fatalf("write policy: %v", err)
	}

	var stdout bytes.Buffer
	if err := run(context.Background(), []string{
		"--generate-key",
		"--private-key", privateKeyPath,
		"--public-key", publicKeyPath,
	}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("generate key: %v", err)
	}
	if err := run(context.Background(), []string{
		"--policy", policyPath,
		"--private-key", privateKeyPath,
		"--signature", signaturePath,
	}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("sign policy: %v", err)
	}

	privateKey := readFile(t, privateKeyPath)
	publicKey := readFile(t, publicKeyPath)
	signature := readFile(t, signaturePath)
	if len(privateKey) != ed25519.PrivateKeySize {
		t.Fatalf("private key length = %d, want %d", len(privateKey), ed25519.PrivateKeySize)
	}
	if len(publicKey) != ed25519.PublicKeySize {
		t.Fatalf("public key length = %d, want %d", len(publicKey), ed25519.PublicKeySize)
	}
	if !ed25519.Verify(publicKey, policyBytes, signature) {
		t.Fatal("signature does not verify policy bytes")
	}
	assertMode(t, privateKeyPath, 0o600)
	assertMode(t, publicKeyPath, 0o644)
	assertMode(t, signaturePath, 0o644)
	if strings.Contains(stdout.String(), string(privateKey)) {
		t.Fatalf("stdout exposed private key bytes: %q", stdout.String())
	}
}

func TestRunRejectsMissingSigningInputs(t *testing.T) {
	t.Parallel()

	err := run(context.Background(), []string{"--policy", "policy.json"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("run succeeded, want missing private key error")
	}
	if !strings.Contains(err.Error(), "private key") {
		t.Fatalf("run error = %q, want private key context", err)
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

func assertMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("%s mode = %o, want %o", path, got, want)
	}
}
