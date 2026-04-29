// Package verify contains reusable verification hooks for downloaded artifacts.
package verify

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/clearsign"
)

// ArtifactInput describes an artifact and any trust material available for it.
//
// Artifact reads the artifact once and runs every check implied by the supplied
// fields. ExpectedSHA256 verifies a literal hex digest. SHA256Sums verifies the
// digest for Name from a SHA256SUMS-style file. Signature verifies an armored
// detached OpenPGP signature and requires Keyring.
type ArtifactInput struct {
	Artifact       io.Reader
	Name           string
	ExpectedSHA256 string
	SHA256Sums     io.Reader
	Signature      io.Reader
	Keyring        io.Reader
}

// HashInput describes an artifact and expected SHA-256 digest.
type HashInput struct {
	Artifact       io.Reader
	ExpectedSHA256 string
	Name           string
}

// SumFileInput describes an artifact checked against a SHA256SUMS-style file.
type SumFileInput struct {
	Artifact io.Reader
	Sums     io.Reader
	Name     string
}

// SignatureInput describes an armored detached OpenPGP signature check.
type SignatureInput struct {
	Artifact  io.Reader
	Signature io.Reader
	Keyring   io.Reader
	Name      string
}

// ClearSignedInput describes a clearsigned OpenPGP message check.
//
// Keyring must contain OpenPGP public keys, either as ASCII armor such as an
// exported "-----BEGIN PGP PUBLIC KEY BLOCK-----" key or as a binary OpenPGP
// public keyring such as Debian's archive keyring. GnuPG keybox databases,
// trust databases, and unrelated PEM formats are not OpenPGP keyrings.
type ClearSignedInput struct {
	Message []byte
	Keyring io.Reader
	Name    string
}

// FileHashInput describes a file and expected SHA-256 digest.
type FileHashInput struct {
	Path           string
	ExpectedSHA256 string
}

// FileSignatureInput describes files for an armored detached signature check.
type FileSignatureInput struct {
	ArtifactPath  string
	SignaturePath string
	KeyringPath   string
}

// Artifact automatically verifies an artifact using all supplied trust
// material.
//
// Verification fails closed: at least one of ExpectedSHA256, SHA256Sums, or
// Signature must be supplied. When multiple checks are supplied, every check
// must pass. Signature verification requires Keyring.
func Artifact(input ArtifactInput) error {
	if input.Artifact == nil {
		return errors.New("artifact is required")
	}
	data, err := io.ReadAll(input.Artifact)
	if err != nil {
		return fmt.Errorf("read %s: %w", displayName(input.Name), err)
	}

	checks := 0
	if input.ExpectedSHA256 != "" {
		checks++
		if err := SHA256(HashInput{
			Artifact:       bytes.NewReader(data),
			ExpectedSHA256: input.ExpectedSHA256,
			Name:           input.Name,
		}); err != nil {
			return err
		}
	}
	if input.SHA256Sums != nil {
		checks++
		if err := SHA256Sums(SumFileInput{
			Artifact: bytes.NewReader(data),
			Sums:     input.SHA256Sums,
			Name:     input.Name,
		}); err != nil {
			return err
		}
	}
	if input.Signature != nil {
		checks++
		if input.Keyring == nil {
			return errors.New("keyring is required for signature verification")
		}
		if err := ArmoredDetachedSignature(SignatureInput{
			Artifact:  bytes.NewReader(data),
			Signature: input.Signature,
			Keyring:   input.Keyring,
			Name:      input.Name,
		}); err != nil {
			return err
		}
	}
	if checks == 0 {
		return errors.New("no verification material supplied")
	}
	return nil
}

// SHA256 verifies that artifact has the expected SHA-256 hex digest.
func SHA256(input HashInput) error {
	if input.Artifact == nil {
		return errors.New("artifact is required")
	}
	if input.ExpectedSHA256 == "" {
		return errors.New("expected sha256 is required")
	}

	hash := sha256.New()
	if _, err := io.Copy(hash, input.Artifact); err != nil {
		return fmt.Errorf("hash %s: %w", displayName(input.Name), err)
	}
	got := hex.EncodeToString(hash.Sum(nil))
	if got != strings.ToLower(input.ExpectedSHA256) {
		return fmt.Errorf("sha256 mismatch for %s", displayName(input.Name))
	}
	return nil
}

// SHA256Sums verifies artifact against an entry in a SHA256SUMS-style file.
func SHA256Sums(input SumFileInput) error {
	if input.Sums == nil {
		return errors.New("sha256 sums are required")
	}
	checksums, err := ParseSHA256Sums(input.Sums)
	if err != nil {
		return err
	}
	want, ok := checksums[input.Name]
	if !ok {
		return fmt.Errorf("checksum for %q not found", input.Name)
	}
	return SHA256(HashInput{
		Artifact:       input.Artifact,
		ExpectedSHA256: want,
		Name:           input.Name,
	})
}

// ParseSHA256Sums parses SHA256SUMS-style data into a map keyed by file name.
func ParseSHA256Sums(reader io.Reader) (map[string]string, error) {
	if reader == nil {
		return nil, errors.New("sha256 sums are required")
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read sha256 sums: %w", err)
	}

	checksums := make(map[string]string)
	for lineNumber, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return nil, fmt.Errorf("parse SHA256SUMS line %d: expected checksum and path", lineNumber+1)
		}
		sum := strings.ToLower(fields[0])
		if len(sum) != sha256.Size*2 {
			return nil, fmt.Errorf("parse SHA256SUMS line %d: invalid sha256 length", lineNumber+1)
		}
		if _, err := hex.DecodeString(sum); err != nil {
			return nil, fmt.Errorf("parse SHA256SUMS line %d: %w", lineNumber+1, err)
		}
		name := strings.TrimPrefix(fields[1], "*")
		name = strings.TrimPrefix(name, "./")
		checksums[name] = sum
	}
	return checksums, nil
}

// ArmoredDetachedSignature verifies an armored or binary detached OpenPGP
// signature.
//
// Keyring must contain OpenPGP public keys, either as ASCII armor such as an
// exported "-----BEGIN PGP PUBLIC KEY BLOCK-----" key or as a binary OpenPGP
// public keyring such as Debian's archive keyring. GnuPG keybox databases,
// trust databases, and unrelated PEM formats are not OpenPGP keyrings.
func ArmoredDetachedSignature(input SignatureInput) error {
	if input.Artifact == nil {
		return errors.New("artifact is required")
	}
	if input.Signature == nil {
		return errors.New("signature is required")
	}
	keyring, err := ReadKeyring(input.Keyring)
	if err != nil {
		return err
	}
	artifact, err := io.ReadAll(input.Artifact)
	if err != nil {
		return fmt.Errorf("read %s: %w", displayName(input.Name), err)
	}
	signature, err := io.ReadAll(input.Signature)
	if err != nil {
		return fmt.Errorf("read signature for %s: %w", displayName(input.Name), err)
	}
	if _, err := openpgp.CheckArmoredDetachedSignature(keyring, bytes.NewReader(artifact), bytes.NewReader(signature), nil); err == nil {
		return nil
	}
	if _, err := openpgp.CheckDetachedSignature(keyring, bytes.NewReader(artifact), bytes.NewReader(signature), nil); err != nil {
		return fmt.Errorf("verify detached signature for %s: %w", displayName(input.Name), err)
	}
	return nil
}

// ClearSigned verifies a clearsigned OpenPGP message and returns trusted plaintext.
func ClearSigned(input ClearSignedInput) ([]byte, error) {
	keyring, err := ReadKeyring(input.Keyring)
	if err != nil {
		return nil, err
	}
	block, _ := clearsign.Decode(input.Message)
	if block == nil {
		return nil, fmt.Errorf("decode clearsigned %s", displayName(input.Name))
	}
	if _, err := block.VerifySignature(keyring, nil); err != nil {
		return nil, fmt.Errorf("verify clearsigned %s: %w", displayName(input.Name), err)
	}
	return block.Plaintext, nil
}

// SHA256File verifies that path has the expected SHA-256 hex digest.
func SHA256File(input FileHashInput) error {
	file, err := os.Open(input.Path)
	if err != nil {
		return fmt.Errorf("open %s: %w", input.Path, err)
	}
	defer func() { _ = file.Close() }()

	return SHA256(HashInput{
		Artifact:       file,
		ExpectedSHA256: input.ExpectedSHA256,
		Name:           input.Path,
	})
}

// ArmoredDetachedSignatureFile verifies files with an armored detached OpenPGP signature.
func ArmoredDetachedSignatureFile(input FileSignatureInput) error {
	keyring, err := os.Open(input.KeyringPath)
	if err != nil {
		return fmt.Errorf("open keyring %s: %w", input.KeyringPath, err)
	}
	defer func() { _ = keyring.Close() }()

	artifact, err := os.Open(input.ArtifactPath)
	if err != nil {
		return fmt.Errorf("open artifact %s: %w", input.ArtifactPath, err)
	}
	defer func() { _ = artifact.Close() }()

	signature, err := os.Open(input.SignaturePath)
	if err != nil {
		return fmt.Errorf("open signature %s: %w", input.SignaturePath, err)
	}
	defer func() { _ = signature.Close() }()

	return ArmoredDetachedSignature(SignatureInput{
		Artifact:  artifact,
		Signature: signature,
		Keyring:   keyring,
		Name:      input.ArtifactPath,
	})
}

// ReadKeyring reads an armored or binary OpenPGP keyring.
//
// Supported inputs are OpenPGP public keys in ASCII armor, including exported
// "-----BEGIN PGP PUBLIC KEY BLOCK-----" data, and binary OpenPGP public
// keyrings such as Debian archive keyring files. GnuPG keybox databases,
// trust databases, and unrelated PEM formats are not accepted.
func ReadKeyring(reader io.Reader) (openpgp.EntityList, error) {
	if reader == nil {
		return nil, errors.New("keyring is required")
	}
	keyringData, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read keyring: %w", err)
	}

	keyring, err := openpgp.ReadArmoredKeyRing(bytes.NewReader(keyringData))
	if err == nil {
		return keyring, nil
	}
	keyring, binaryErr := openpgp.ReadKeyRing(bytes.NewReader(keyringData))
	if binaryErr == nil {
		return keyring, nil
	}
	return nil, fmt.Errorf("read OpenPGP keyring: %w", errors.Join(err, binaryErr))
}

func displayName(name string) string {
	if name == "" {
		return "artifact"
	}
	return name
}
