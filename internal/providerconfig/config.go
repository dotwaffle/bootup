// Package providerconfig loads operator-supplied provider runtime config.
package providerconfig

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/dotwaffle/bootup/verify"
)

// Config contains runtime configuration for compiled-in providers.
type Config struct {
	Debian DebianConfig
	Ubuntu UbuntuConfig
}

// DebianConfig contains runtime configuration for the Debian provider.
type DebianConfig struct {
	MirrorURL string
	Keyring   []byte
}

// UbuntuConfig contains runtime configuration for the Ubuntu provider.
type UbuntuConfig struct {
	ReleaseURL   string
	Keyring      []byte
	KernelSHA256 string
	InitrdSHA256 string
}

type fileConfig struct {
	Providers map[string]json.RawMessage `json:"providers"`
}

type debianFileConfig struct {
	MirrorURL   string `json:"mirror_url"`
	KeyringPath string `json:"keyring_path"`
}

type ubuntuFileConfig struct {
	ReleaseURL   string `json:"release_url"`
	KeyringPath  string `json:"keyring_path"`
	KernelSHA256 string `json:"kernel_sha256"`
	InitrdSHA256 string `json:"initrd_sha256"`
}

// LoadFile reads and validates provider runtime configuration from path.
func LoadFile(path string) (Config, error) {
	if strings.TrimSpace(path) == "" {
		return Config{}, errors.New("provider config path is required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read provider config %s: %w", path, err)
	}

	var file fileConfig
	if err := decodeStrict(data, &file); err != nil {
		return Config{}, fmt.Errorf("parse provider config %s: %w", path, err)
	}

	var config Config
	for id, raw := range file.Providers {
		switch id {
		case "debian":
			debianConfig, err := loadDebian(raw)
			if err != nil {
				return Config{}, fmt.Errorf("load Debian provider config: %w", err)
			}
			config.Debian = debianConfig
		case "ubuntu":
			ubuntuConfig, err := loadUbuntu(raw)
			if err != nil {
				return Config{}, fmt.Errorf("load Ubuntu provider config: %w", err)
			}
			config.Ubuntu = ubuntuConfig
		default:
			return Config{}, fmt.Errorf("unknown provider %q", id)
		}
	}
	return config, nil
}

func loadDebian(raw json.RawMessage) (DebianConfig, error) {
	var file debianFileConfig
	if err := decodeStrict(raw, &file); err != nil {
		return DebianConfig{}, err
	}
	if err := validateHTTPURL("mirror_url", file.MirrorURL); err != nil {
		return DebianConfig{}, err
	}
	keyring, err := loadKeyring(file.KeyringPath)
	if err != nil {
		return DebianConfig{}, err
	}
	return DebianConfig{
		MirrorURL: strings.TrimRight(file.MirrorURL, "/"),
		Keyring:   keyring,
	}, nil
}

func loadUbuntu(raw json.RawMessage) (UbuntuConfig, error) {
	var file ubuntuFileConfig
	if err := decodeStrict(raw, &file); err != nil {
		return UbuntuConfig{}, err
	}
	if err := validateHTTPURL("release_url", file.ReleaseURL); err != nil {
		return UbuntuConfig{}, err
	}
	if err := validateSHA256Pins(file.KernelSHA256, file.InitrdSHA256); err != nil {
		return UbuntuConfig{}, err
	}
	keyring, err := loadKeyring(file.KeyringPath)
	if err != nil {
		return UbuntuConfig{}, err
	}
	return UbuntuConfig{
		ReleaseURL:   strings.TrimRight(file.ReleaseURL, "/"),
		Keyring:      keyring,
		KernelSHA256: strings.ToLower(file.KernelSHA256),
		InitrdSHA256: strings.ToLower(file.InitrdSHA256),
	}, nil
}

func decodeStrict(data []byte, output any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(output); err != nil {
		return err
	}
	var extra struct{}
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func validateHTTPURL(field string, value string) error {
	if value == "" {
		return nil
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("parse %s: %w", field, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("%s must use http or https", field)
	}
	if parsed.Host == "" {
		return fmt.Errorf("%s must include host", field)
	}
	return nil
}

func validateSHA256Pins(kernelSHA256 string, initrdSHA256 string) error {
	if (kernelSHA256 == "") != (initrdSHA256 == "") {
		return errors.New("kernel_sha256 and initrd_sha256 must be supplied together")
	}
	if kernelSHA256 == "" {
		return nil
	}
	if err := validateSHA256("kernel_sha256", kernelSHA256); err != nil {
		return err
	}
	return validateSHA256("initrd_sha256", initrdSHA256)
}

func validateSHA256(field string, value string) error {
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != 32 {
		return fmt.Errorf("%s must be a 64-character SHA-256 hex digest", field)
	}
	return nil
}

func loadKeyring(path string) ([]byte, error) {
	if path == "" {
		return nil, nil
	}
	keyring, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read keyring %s: %w", path, err)
	}
	if _, err := verify.ReadKeyring(bytes.NewReader(keyring)); err != nil {
		return nil, fmt.Errorf("validate keyring %s: %w", path, err)
	}
	return bytes.Clone(keyring), nil
}
