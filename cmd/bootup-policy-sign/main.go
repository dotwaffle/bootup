package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
)

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "bootup-policy-sign: %v\n", err)
		os.Exit(1)
	}
}

func run(_ context.Context, args []string, stdout io.Writer, stderr io.Writer) error {
	flags := flag.NewFlagSet("bootup-policy-sign", flag.ContinueOnError)
	flags.SetOutput(stderr)

	generateKey := flags.Bool("generate-key", false, "generate an Ed25519 private/public key pair")
	privateKeyPath := flags.String("private-key", "", "raw Ed25519 private key path")
	publicKeyPath := flags.String("public-key", "", "raw Ed25519 public key path")
	policyPath := flags.String("policy", "", "policy JSON file to sign")
	signaturePath := flags.String("signature", "", "detached signature output path")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	if *generateKey {
		if *policyPath != "" || *signaturePath != "" {
			return errors.New("--generate-key cannot be combined with --policy or --signature")
		}
		return generatePolicyKeyPair(generateKeyInput{
			PrivateKeyPath: *privateKeyPath,
			PublicKeyPath:  *publicKeyPath,
			Stdout:         stdout,
		})
	}
	return signPolicy(signPolicyInput{
		PolicyPath:     *policyPath,
		PrivateKeyPath: *privateKeyPath,
		SignaturePath:  *signaturePath,
		Stdout:         stdout,
	})
}

type generateKeyInput struct {
	PrivateKeyPath string
	PublicKeyPath  string
	Stdout         io.Writer
}

func generatePolicyKeyPair(input generateKeyInput) error {
	if input.PrivateKeyPath == "" {
		return errors.New("private key output path is required")
	}
	if input.PublicKeyPath == "" {
		return errors.New("public key output path is required")
	}
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("generate Ed25519 policy key: %w", err)
	}
	if err := writeNewFile(input.PrivateKeyPath, privateKey, 0o600); err != nil {
		return fmt.Errorf("write private key: %w", err)
	}
	if err := writeNewFile(input.PublicKeyPath, publicKey, 0o644); err != nil {
		return fmt.Errorf("write public key: %w", err)
	}
	_, err = fmt.Fprintf(input.Stdout, "wrote %s\nwrote %s\n", input.PrivateKeyPath, input.PublicKeyPath)
	return err
}

type signPolicyInput struct {
	PolicyPath     string
	PrivateKeyPath string
	SignaturePath  string
	Stdout         io.Writer
}

func signPolicy(input signPolicyInput) error {
	if input.PolicyPath == "" {
		return errors.New("policy path is required")
	}
	if input.PrivateKeyPath == "" {
		return errors.New("private key path is required")
	}
	if input.SignaturePath == "" {
		return errors.New("signature output path is required")
	}
	policyBytes, err := os.ReadFile(input.PolicyPath)
	if err != nil {
		return fmt.Errorf("read policy: %w", err)
	}
	privateKey, err := readHexOrRawFile(input.PrivateKeyPath)
	if err != nil {
		return fmt.Errorf("read private key: %w", err)
	}
	if len(privateKey) != ed25519.PrivateKeySize {
		return fmt.Errorf("private key length is %d bytes, want %d", len(privateKey), ed25519.PrivateKeySize)
	}
	signature := ed25519.Sign(privateKey, policyBytes)
	if err := writeNewFile(input.SignaturePath, signature, 0o644); err != nil {
		return fmt.Errorf("write signature: %w", err)
	}
	_, err = fmt.Fprintf(input.Stdout, "wrote %s\n", input.SignaturePath)
	return err
}

func writeNewFile(path string, data []byte, mode os.FileMode) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Chmod(mode); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func readHexOrRawFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	trimmed := bytes.TrimSpace(data)
	decoded, err := hex.DecodeString(string(trimmed))
	if err == nil && len(decoded) > 0 {
		return decoded, nil
	}
	return data, nil
}
