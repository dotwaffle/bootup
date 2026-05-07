// Package policy loads and validates data-only dynamic boot decisions.
package policy

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/dotwaffle/bootup/internal/provider"
)

const (
	schemaVersion          = 1
	defaultPolicyMaxBytes  = 1 << 20
	defaultPolicyCacheMode = 0o644
)

// ErrInvalidPolicy is returned when a policy decision cannot be trusted or
// used.
var ErrInvalidPolicy = errors.New("invalid policy")

// Decision is an authenticated data-only policy result.
type Decision struct {
	SchemaVersion int               `json:"schema_version"`
	DecisionID    string            `json:"decision_id"`
	TargetID      string            `json:"target_id"`
	Options       map[string]string `json:"options,omitzero"`
	SecretRefs    map[string]string `json:"secret_refs,omitzero"`
	PublishedAt   *time.Time        `json:"published_at,omitzero"`
	ExpiresAt     *time.Time        `json:"expires_at,omitzero"`
}

// Trust describes operator-supplied policy authenticity checks.
type Trust struct {
	Ed25519Signature []byte
	Ed25519PublicKey []byte
}

// LoadOptions configures signed local policy loading.
type LoadOptions struct {
	Path          string
	Trust         Trust
	MaxBytes      int64
	Now           func() time.Time
	MaxAge        time.Duration
	CachePath     string
	CacheFallback bool
}

// Selection is a validated policy choice ready for provider planning.
type Selection struct {
	Target     provider.Target
	Options    []provider.SelectedOption
	SecretRefs []provider.SecretRef
}

// ValidateInput contains the inventory and secret store for decision
// validation.
type ValidateInput struct {
	Decision Decision
	Targets  []provider.Target
	Secrets  provider.SecretStore
}

// LoadFile reads, authenticates, parses, and freshness-checks a local policy.
func LoadFile(options LoadOptions) (Decision, error) {
	if !options.Trust.configured() {
		return Decision{}, fmt.Errorf("%w: policy trust configuration is required", ErrInvalidPolicy)
	}
	data, err := readPolicyFile(options.Path, options.MaxBytes)
	if err != nil {
		if options.CacheFallback && options.CachePath != "" {
			decision, cacheErr := loadPolicyCache(options)
			if cacheErr != nil {
				return Decision{}, fmt.Errorf("%w: load policy cache after source failure: %w", err, cacheErr)
			}
			return decision, nil
		}
		return Decision{}, err
	}
	decision, err := parseTrustedDecision(data, options)
	if err != nil {
		return Decision{}, err
	}
	if options.CachePath != "" {
		if err := writePolicyCache(options.CachePath, data); err != nil {
			return Decision{}, err
		}
	}
	return decision, nil
}

// Validate checks a policy decision against the current target inventory.
func Validate(input ValidateInput) (Selection, error) {
	target, ok := targetByID(input.Targets, input.Decision.TargetID)
	if !ok {
		return Selection{}, fmt.Errorf("%w: policy target %q is not in the current inventory", ErrInvalidPolicy, input.Decision.TargetID)
	}
	options := selectedOptions(input.Decision.Options)
	if err := provider.ValidateSelectedOptions(target, options); err != nil {
		return Selection{}, fmt.Errorf("%w: %w", ErrInvalidPolicy, err)
	}
	secretRefs := selectedSecretRefs(input.Decision.SecretRefs)
	resolvedRefs, err := provider.ResolveSecretRefs(target, secretRefs, input.Secrets)
	if err != nil {
		return Selection{}, fmt.Errorf("%w: %w", ErrInvalidPolicy, err)
	}
	return Selection{
		Target:     target,
		Options:    options,
		SecretRefs: resolvedRefs,
	}, nil
}

func parseTrustedDecision(data []byte, options LoadOptions) (Decision, error) {
	if err := verifyTrust(data, options.Trust); err != nil {
		return Decision{}, err
	}
	decision, err := parseDecision(data)
	if err != nil {
		return Decision{}, err
	}
	if err := validateFreshness(decision, options); err != nil {
		return Decision{}, err
	}
	return decision, nil
}

func parseDecision(data []byte) (Decision, error) {
	var decision Decision
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decision); err != nil {
		return Decision{}, fmt.Errorf("%w: decode policy decision: %w", ErrInvalidPolicy, err)
	}
	if err := decoder.Decode(new(struct{})); !errors.Is(err, io.EOF) {
		return Decision{}, fmt.Errorf("%w: decode policy decision: multiple JSON values", ErrInvalidPolicy)
	}
	if decision.SchemaVersion != schemaVersion {
		return Decision{}, fmt.Errorf("%w: unsupported policy schema version %d", ErrInvalidPolicy, decision.SchemaVersion)
	}
	if decision.DecisionID == "" {
		return Decision{}, fmt.Errorf("%w: policy decision_id is required", ErrInvalidPolicy)
	}
	if decision.TargetID == "" {
		return Decision{}, fmt.Errorf("%w: policy target_id is required", ErrInvalidPolicy)
	}
	return decision, nil
}

func validateFreshness(decision Decision, options LoadOptions) error {
	now := time.Now
	if options.Now != nil {
		now = options.Now
	}
	currentTime := now()
	if decision.ExpiresAt == nil && (options.MaxAge <= 0 || decision.PublishedAt == nil) {
		return fmt.Errorf("%w: policy freshness metadata is required", ErrInvalidPolicy)
	}
	if decision.ExpiresAt != nil && !decision.ExpiresAt.After(currentTime) {
		return fmt.Errorf("%w: policy decision expired at %s", ErrInvalidPolicy, decision.ExpiresAt.Format(time.RFC3339))
	}
	if options.MaxAge > 0 {
		if decision.PublishedAt == nil {
			return fmt.Errorf("%w: policy published_at is required for maximum age validation", ErrInvalidPolicy)
		}
		if decision.PublishedAt.After(currentTime) {
			return fmt.Errorf("%w: policy published_at is in the future", ErrInvalidPolicy)
		}
		if currentTime.Sub(*decision.PublishedAt) > options.MaxAge {
			return fmt.Errorf("%w: policy decision exceeds maximum age %s", ErrInvalidPolicy, options.MaxAge)
		}
	}
	return nil
}

func readPolicyFile(path string, maxBytes int64) ([]byte, error) {
	if path == "" {
		return nil, fmt.Errorf("%w: policy file path is required", ErrInvalidPolicy)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("%w: read policy file: %w", ErrInvalidPolicy, err)
	}
	defer func() { _ = file.Close() }()
	return readPolicyBytes(file, maxBytes)
}

func readPolicyBytes(reader io.Reader, maxBytes int64) ([]byte, error) {
	if maxBytes <= 0 {
		maxBytes = defaultPolicyMaxBytes
	}
	data, err := io.ReadAll(io.LimitReader(reader, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("%w: read policy decision: %w", ErrInvalidPolicy, err)
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("%w: policy decision is too large", ErrInvalidPolicy)
	}
	return data, nil
}

func loadPolicyCache(options LoadOptions) (Decision, error) {
	data, err := readPolicyFile(options.CachePath, options.MaxBytes)
	if err != nil {
		return Decision{}, err
	}
	return parseTrustedDecision(data, options)
}

func writePolicyCache(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("%w: create policy cache directory: %w", ErrInvalidPolicy, err)
	}
	file, err := os.CreateTemp(dir, ".bootup-policy-cache-*")
	if err != nil {
		return fmt.Errorf("%w: create policy cache temp file: %w", ErrInvalidPolicy, err)
	}
	tempPath := file.Name()
	defer func() { _ = os.Remove(tempPath) }()
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return fmt.Errorf("%w: write policy cache temp file: %w", ErrInvalidPolicy, err)
	}
	if err := file.Chmod(defaultPolicyCacheMode); err != nil {
		_ = file.Close()
		return fmt.Errorf("%w: chmod policy cache temp file: %w", ErrInvalidPolicy, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("%w: close policy cache temp file: %w", ErrInvalidPolicy, err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("%w: replace policy cache: %w", ErrInvalidPolicy, err)
	}
	return nil
}

func verifyTrust(data []byte, trust Trust) error {
	if !trust.configured() {
		return fmt.Errorf("%w: policy trust configuration is required", ErrInvalidPolicy)
	}
	if len(trust.Ed25519Signature) != ed25519.SignatureSize {
		return fmt.Errorf("%w: policy Ed25519 signature length is %d bytes, want %d", ErrInvalidPolicy, len(trust.Ed25519Signature), ed25519.SignatureSize)
	}
	if len(trust.Ed25519PublicKey) != ed25519.PublicKeySize {
		return fmt.Errorf("%w: policy Ed25519 public key length is %d bytes, want %d", ErrInvalidPolicy, len(trust.Ed25519PublicKey), ed25519.PublicKeySize)
	}
	if !ed25519.Verify(trust.Ed25519PublicKey, data, trust.Ed25519Signature) {
		return fmt.Errorf("%w: policy Ed25519 signature verification failed", ErrInvalidPolicy)
	}
	return nil
}

func (trust Trust) configured() bool {
	return len(trust.Ed25519Signature) != 0 || len(trust.Ed25519PublicKey) != 0
}

func targetByID(targets []provider.Target, id string) (provider.Target, bool) {
	for _, target := range targets {
		if target.ID == id {
			return target, true
		}
	}
	return provider.Target{}, false
}

func selectedOptions(values map[string]string) []provider.SelectedOption {
	if len(values) == 0 {
		return nil
	}
	ids := make([]string, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	selected := make([]provider.SelectedOption, 0, len(ids))
	for _, id := range ids {
		selected = append(selected, provider.SelectedOption{ID: id, Value: values[id]})
	}
	return selected
}

func selectedSecretRefs(values map[string]string) []provider.SecretRef {
	if len(values) == 0 {
		return nil
	}
	ids := make([]string, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	selected := make([]provider.SecretRef, 0, len(ids))
	for _, id := range ids {
		selected = append(selected, provider.SecretRef{ID: id, InputID: values[id]})
	}
	return selected
}
