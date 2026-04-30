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
	"time"

	"github.com/dotwaffle/bootup/internal/provider"
	"github.com/dotwaffle/bootup/verify"
)

// Config contains runtime configuration for compiled-in providers.
type Config struct {
	Debian DebianConfig
	Ubuntu UbuntuConfig
	Fedora FedoraConfig
}

// DebianConfig contains runtime configuration for the Debian provider.
type DebianConfig struct {
	MirrorURL        string
	DiscoveryURL     string
	DiscoveryTimeout time.Duration
	Keyring          []byte
	Lifecycle        map[string]provider.LifecycleEntry
}

// UbuntuConfig contains runtime configuration for the Ubuntu provider.
type UbuntuConfig struct {
	ReleaseURL       string
	DiscoveryURL     string
	DiscoveryTimeout time.Duration
	Keyring          []byte
	KernelSHA256     string
	InitrdSHA256     string
	Lifecycle        map[string]provider.LifecycleEntry
}

// FedoraConfig contains runtime configuration for the Fedora provider.
type FedoraConfig struct {
	ReleaseURL   string
	KernelSHA256 string
	InitrdSHA256 string
}

type fileConfig struct {
	Providers map[string]json.RawMessage `json:"providers"`
}

type debianFileConfig struct {
	MirrorURL        string                             `json:"mirror_url"`
	DiscoveryURL     string                             `json:"discovery_url"`
	DiscoveryTimeout string                             `json:"discovery_timeout"`
	KeyringPath      string                             `json:"keyring_path"`
	Lifecycle        map[string]provider.LifecycleEntry `json:"lifecycle"`
}

type ubuntuFileConfig struct {
	ReleaseURL       string                             `json:"release_url"`
	DiscoveryURL     string                             `json:"discovery_url"`
	DiscoveryTimeout string                             `json:"discovery_timeout"`
	KeyringPath      string                             `json:"keyring_path"`
	KernelSHA256     string                             `json:"kernel_sha256"`
	InitrdSHA256     string                             `json:"initrd_sha256"`
	Lifecycle        map[string]provider.LifecycleEntry `json:"lifecycle"`
}

type fedoraFileConfig struct {
	ReleaseURL   string `json:"release_url"`
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
		case "fedora":
			fedoraConfig, err := loadFedora(raw)
			if err != nil {
				return Config{}, fmt.Errorf("load Fedora provider config: %w", err)
			}
			config.Fedora = fedoraConfig
		default:
			return Config{}, fmt.Errorf("unknown provider %q", id)
		}
	}
	return config, nil
}

func loadFedora(raw json.RawMessage) (FedoraConfig, error) {
	var file fedoraFileConfig
	if err := decodeStrict(raw, &file); err != nil {
		return FedoraConfig{}, err
	}
	if err := validateHTTPURL("release_url", file.ReleaseURL); err != nil {
		return FedoraConfig{}, err
	}
	if err := validateSHA256Pins(file.KernelSHA256, file.InitrdSHA256); err != nil {
		return FedoraConfig{}, err
	}
	return FedoraConfig{
		ReleaseURL:   strings.TrimRight(file.ReleaseURL, "/"),
		KernelSHA256: strings.ToLower(file.KernelSHA256),
		InitrdSHA256: strings.ToLower(file.InitrdSHA256),
	}, nil
}

func loadDebian(raw json.RawMessage) (DebianConfig, error) {
	var file debianFileConfig
	if err := decodeStrict(raw, &file); err != nil {
		return DebianConfig{}, err
	}
	if err := validateHTTPURL("mirror_url", file.MirrorURL); err != nil {
		return DebianConfig{}, err
	}
	if err := validateHTTPURL("discovery_url", file.DiscoveryURL); err != nil {
		return DebianConfig{}, err
	}
	discoveryTimeout, err := parseDuration("discovery_timeout", file.DiscoveryTimeout)
	if err != nil {
		return DebianConfig{}, err
	}
	lifecycle, err := validateLifecycle(file.Lifecycle)
	if err != nil {
		return DebianConfig{}, err
	}
	keyring, err := loadKeyring(file.KeyringPath)
	if err != nil {
		return DebianConfig{}, err
	}
	return DebianConfig{
		MirrorURL:        strings.TrimRight(file.MirrorURL, "/"),
		DiscoveryURL:     strings.TrimRight(file.DiscoveryURL, "/"),
		DiscoveryTimeout: discoveryTimeout,
		Keyring:          keyring,
		Lifecycle:        lifecycle,
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
	if err := validateHTTPURL("discovery_url", file.DiscoveryURL); err != nil {
		return UbuntuConfig{}, err
	}
	discoveryTimeout, err := parseDuration("discovery_timeout", file.DiscoveryTimeout)
	if err != nil {
		return UbuntuConfig{}, err
	}
	if err := validateSHA256Pins(file.KernelSHA256, file.InitrdSHA256); err != nil {
		return UbuntuConfig{}, err
	}
	lifecycle, err := validateLifecycle(file.Lifecycle)
	if err != nil {
		return UbuntuConfig{}, err
	}
	keyring, err := loadKeyring(file.KeyringPath)
	if err != nil {
		return UbuntuConfig{}, err
	}
	return UbuntuConfig{
		ReleaseURL:       strings.TrimRight(file.ReleaseURL, "/"),
		DiscoveryURL:     strings.TrimRight(file.DiscoveryURL, "/"),
		DiscoveryTimeout: discoveryTimeout,
		Keyring:          keyring,
		KernelSHA256:     strings.ToLower(file.KernelSHA256),
		InitrdSHA256:     strings.ToLower(file.InitrdSHA256),
		Lifecycle:        lifecycle,
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

func parseDuration(field string, value string) (time.Duration, error) {
	if value == "" {
		return 0, nil
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", field, err)
	}
	if duration <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", field)
	}
	return duration, nil
}

func validateLifecycle(entries map[string]provider.LifecycleEntry) (map[string]provider.LifecycleEntry, error) {
	if len(entries) == 0 {
		return map[string]provider.LifecycleEntry{}, nil
	}
	out := make(map[string]provider.LifecycleEntry, len(entries))
	for release, entry := range entries {
		if strings.TrimSpace(release) == "" {
			return nil, errors.New("lifecycle release must not be empty")
		}
		if strings.TrimSpace(release) != release {
			return nil, fmt.Errorf("lifecycle release %q has surrounding whitespace", release)
		}
		if strings.TrimSpace(entry.Source) != entry.Source {
			return nil, fmt.Errorf("lifecycle source for %s has surrounding whitespace", release)
		}
		if entry.Source == "" {
			return nil, fmt.Errorf("lifecycle source for %s is required", release)
		}
		if strings.TrimSpace(entry.Date) != entry.Date {
			return nil, fmt.Errorf("lifecycle date for %s has surrounding whitespace", release)
		}
		switch entry.Status {
		case provider.LifecycleSupported, provider.LifecycleObsolete, provider.LifecycleEOL, provider.LifecycleUnknown:
		case "":
			return nil, fmt.Errorf("lifecycle status for %s is required", release)
		default:
			return nil, fmt.Errorf("lifecycle status for %s is invalid", release)
		}
		if entry.Date != "" {
			if _, err := time.Parse(time.DateOnly, entry.Date); err != nil {
				return nil, fmt.Errorf("lifecycle date for %s must use YYYY-MM-DD", release)
			}
		}
		out[release] = entry
	}
	return out, nil
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
