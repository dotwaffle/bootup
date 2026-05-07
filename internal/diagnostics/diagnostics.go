// Package diagnostics writes opt-in failure bundles for bootup runs.
package diagnostics

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dotwaffle/bootup/internal/provider"
)

const (
	schemaVersion = 1
	redactedValue = "<redacted>"
)

// CatalogPosture describes how the active catalog was sourced and checked
// without storing catalog URLs, paths, or trust bytes.
type CatalogPosture struct {
	Source         string `json:"source"`
	LocalPathSet   bool   `json:"local_path_set,omitzero"`
	HostedURLSet   bool   `json:"hosted_url_set,omitzero"`
	SHA256         bool   `json:"sha256,omitzero"`
	Ed25519        bool   `json:"ed25519,omitzero"`
	SignatureFiles bool   `json:"signature_files,omitzero"`
	Freshness      bool   `json:"freshness,omitzero"`
	Cache          bool   `json:"cache,omitzero"`
	CacheFallback  bool   `json:"cache_fallback,omitzero"`
	MaxBytes       bool   `json:"max_bytes,omitzero"`
}

// ProviderConfigPosture describes whether a provider config path was supplied
// without storing the path or file contents.
type ProviderConfigPosture struct {
	PathSet bool `json:"path_set"`
}

// Summary is the structured diagnostics metadata written to summary.json.
type Summary struct {
	SchemaVersion     int                   `json:"schema_version"`
	CreatedAt         string                `json:"created_at"`
	Mode              string                `json:"mode,omitzero"`
	TargetID          string                `json:"target_id,omitzero"`
	DiscoveryFamilyID string                `json:"discovery_family_id,omitzero"`
	SelectedOptionIDs []string              `json:"selected_option_ids,omitzero"`
	SecretInputIDs    []string              `json:"secret_input_ids,omitzero"`
	SecretRefIDs      []string              `json:"secret_ref_ids,omitzero"`
	Catalog           CatalogPosture        `json:"catalog"`
	ProviderConfig    ProviderConfigPosture `json:"provider_config"`
	Error             string                `json:"error,omitzero"`
}

// SummaryInput contains run metadata for a redacted diagnostics summary.
type SummaryInput struct {
	Mode              string
	TargetID          string
	DiscoveryFamilyID string
	SelectedOptions   []provider.SelectedOption
	SecretInputIDs    []string
	SecretRefIDs      []string
	Catalog           CatalogPosture
	ProviderConfig    ProviderConfigPosture
	Error             error
	RedactValues      []string
}

// BuildSummary converts run metadata into the redacted form persisted in a
// diagnostics bundle.
func BuildSummary(input SummaryInput) Summary {
	optionIDs := make([]string, 0, len(input.SelectedOptions))
	redactions := make([]string, 0, len(input.SelectedOptions)+len(input.RedactValues))
	for _, option := range input.SelectedOptions {
		optionIDs = append(optionIDs, option.ID)
		redactions = append(redactions, option.Value)
	}
	redactions = append(redactions, input.RedactValues...)

	return Summary{
		SchemaVersion:     schemaVersion,
		Mode:              input.Mode,
		TargetID:          input.TargetID,
		DiscoveryFamilyID: input.DiscoveryFamilyID,
		SelectedOptionIDs: optionIDs,
		SecretInputIDs:    append([]string(nil), input.SecretInputIDs...),
		SecretRefIDs:      append([]string(nil), input.SecretRefIDs...),
		Catalog:           input.Catalog,
		ProviderConfig:    input.ProviderConfig,
		Error:             redactError(input.Error, redactions),
	}
}

func redactError(err error, values []string) string {
	if err == nil {
		return ""
	}
	text := err.Error()
	for _, value := range values {
		if value == "" {
			continue
		}
		text = strings.ReplaceAll(text, value, redactedValue)
	}
	return text
}

// Bundle is the diagnostics payload written under RootDir.
type Bundle struct {
	RootDir   string
	CreatedAt time.Time
	Summary   Summary
	Stdout    []byte
	Stderr    []byte
}

// WriteBundle writes a timestamped diagnostics bundle and returns its
// directory.
func WriteBundle(bundle Bundle) (string, error) {
	if strings.TrimSpace(bundle.RootDir) == "" {
		return "", errors.New("diagnostics root directory is required")
	}

	createdAt := bundle.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	createdAt = createdAt.UTC()

	summary := bundle.Summary
	if summary.SchemaVersion == 0 {
		summary.SchemaVersion = schemaVersion
	}
	summary.CreatedAt = createdAt.Format(time.RFC3339)

	dir, err := createBundleDir(bundle.RootDir, createdAt)
	if err != nil {
		return "", err
	}
	if err := writeJSONFile(filepath.Join(dir, "summary.json"), summary); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dir, "stdout.txt"), bundle.Stdout, 0o600); err != nil {
		return "", fmt.Errorf("write diagnostics stdout: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "stderr.txt"), bundle.Stderr, 0o600); err != nil {
		return "", fmt.Errorf("write diagnostics stderr: %w", err)
	}
	return dir, nil
}

func createBundleDir(root string, createdAt time.Time) (string, error) {
	baseName := "bootup-" + createdAt.Format("20060102T150405Z")
	for attempt := range 1000 {
		name := baseName
		if attempt > 0 {
			name = fmt.Sprintf("%s-%03d", baseName, attempt)
		}
		path := filepath.Join(root, name)
		if err := os.MkdirAll(path, 0o700); err != nil {
			return "", fmt.Errorf("create diagnostics directory %s: %w", path, err)
		}
		info, err := os.Stat(path)
		if err != nil {
			return "", fmt.Errorf("stat diagnostics directory %s: %w", path, err)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("create diagnostics directory %s: not a directory", path)
		}
		if attempt == 0 || isEmptyDir(path) {
			return path, nil
		}
	}
	return "", fmt.Errorf("create diagnostics directory %s: too many collisions", filepath.Join(root, baseName))
}

func isEmptyDir(path string) bool {
	entries, err := os.ReadDir(path)
	return err == nil && len(entries) == 0
}

func writeJSONFile(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode diagnostics summary: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write diagnostics summary: %w", err)
	}
	return nil
}

// Capture tees stdout and stderr to their original writers while buffering the
// same bytes for a diagnostics bundle.
type Capture struct {
	stdout bytes.Buffer
	stderr bytes.Buffer
}

// NewCapture creates a diagnostics capture wrapper for stdout and stderr.
func NewCapture() *Capture {
	return &Capture{}
}

// Stdout returns a writer that copies writes to output and the stdout buffer.
func (c *Capture) Stdout(output io.Writer) io.Writer {
	return io.MultiWriter(output, &c.stdout)
}

// Stderr returns a writer that copies writes to output and the stderr buffer.
func (c *Capture) Stderr(output io.Writer) io.Writer {
	return io.MultiWriter(output, &c.stderr)
}

// StdoutBytes returns captured stdout bytes.
func (c *Capture) StdoutBytes() []byte {
	return append([]byte(nil), c.stdout.Bytes()...)
}

// StderrBytes returns captured stderr bytes.
func (c *Capture) StderrBytes() []byte {
	return append([]byte(nil), c.stderr.Bytes()...)
}
