package catalog

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

const defaultHostedCatalogMaxBytes int64 = 1 << 20

// HostedOptions configures URL-hosted static catalog loading.
type HostedOptions struct {
	URL              string
	HTTPClient       *http.Client
	Trust            HostedTrust
	ProviderIDs      []string
	MaxBytes         int64
	Now              func() time.Time
	MaxAge           time.Duration
	RequireFreshness bool
	CachePath        string
	CacheFallback    bool
}

// HostedTrust describes operator-supplied trust checks for a hosted catalog.
type HostedTrust struct {
	SHA256           string
	Ed25519Signature []byte
	Ed25519PublicKey []byte
}

// LoadHosted fetches, authenticates, parses, and validates a hosted catalog.
func LoadHosted(ctx context.Context, options HostedOptions) (Document, error) {
	if !options.Trust.configured() {
		return Document{}, fmt.Errorf("%w: hosted catalog trust configuration is required", ErrInvalidCatalog)
	}
	data, err := fetchHostedCatalog(ctx, options)
	if err != nil {
		if options.CacheFallback && options.CachePath != "" && !errors.Is(err, ErrInvalidCatalog) {
			doc, cacheErr := loadHostedCache(options)
			if cacheErr != nil {
				return Document{}, fmt.Errorf("load hosted catalog cache after fetch failure: %w", cacheErr)
			}
			return doc, nil
		}
		return Document{}, err
	}
	doc, err := parseHostedCatalog(data, options)
	if err != nil {
		return Document{}, err
	}
	if options.CachePath != "" {
		if err := writeHostedCache(options.CachePath, data); err != nil {
			return Document{}, err
		}
	}
	return doc, nil
}

func parseHostedCatalog(data []byte, options HostedOptions) (Document, error) {
	if err := VerifyHostedTrust(data, options.Trust); err != nil {
		return Document{}, err
	}
	doc, err := Parse(data, options.ProviderIDs)
	if err != nil {
		return Document{}, fmt.Errorf("parse hosted catalog %s: %w", options.URL, err)
	}
	if err := validateHostedFreshness(doc, options); err != nil {
		return Document{}, err
	}
	return doc, nil
}

// VerifyHostedTrust authenticates hosted catalog bytes before parsing.
func VerifyHostedTrust(data []byte, trust HostedTrust) error {
	if !trust.configured() {
		return fmt.Errorf("%w: hosted catalog trust configuration is required", ErrInvalidCatalog)
	}
	if trust.SHA256 != "" {
		if err := verifySHA256(data, trust.SHA256); err != nil {
			return err
		}
	}
	if len(trust.Ed25519Signature) > 0 || len(trust.Ed25519PublicKey) > 0 {
		if err := verifyEd25519(data, trust.Ed25519Signature, trust.Ed25519PublicKey); err != nil {
			return err
		}
	}
	return nil
}

func (trust HostedTrust) configured() bool {
	return trust.SHA256 != "" || len(trust.Ed25519Signature) > 0 || len(trust.Ed25519PublicKey) > 0
}

func fetchHostedCatalog(ctx context.Context, options HostedOptions) ([]byte, error) {
	parsedURL, err := url.Parse(options.URL)
	if err != nil {
		return nil, fmt.Errorf("%w: parse hosted catalog URL: %w", ErrInvalidCatalog, err)
	}
	if parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("%w: hosted catalog URL must use https", ErrInvalidCatalog)
	}
	client := options.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, options.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("new hosted catalog request: %w", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("fetch hosted catalog %s: %w", options.URL, err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("GET %s: %s", options.URL, http.StatusText(response.StatusCode))
	}
	return readHostedCatalogBytes(response.Body, options.MaxBytes)
}

func readHostedCatalogBytes(reader io.Reader, maxBytes int64) ([]byte, error) {
	if maxBytes <= 0 {
		maxBytes = defaultHostedCatalogMaxBytes
	}
	data, err := io.ReadAll(io.LimitReader(reader, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read hosted catalog: %w", err)
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("%w: hosted catalog response is too large", ErrInvalidCatalog)
	}
	return data, nil
}

func validateHostedFreshness(doc Document, options HostedOptions) error {
	now := time.Now
	if options.Now != nil {
		now = options.Now
	}
	currentTime := now()
	if options.RequireFreshness && doc.PublishedAt == nil && doc.ExpiresAt == nil {
		return fmt.Errorf("%w: hosted catalog freshness metadata is required", ErrInvalidCatalog)
	}
	if doc.ExpiresAt != nil && !doc.ExpiresAt.After(currentTime) {
		return fmt.Errorf("%w: hosted catalog expired at %s", ErrInvalidCatalog, doc.ExpiresAt.Format(time.RFC3339))
	}
	if options.MaxAge > 0 {
		if doc.PublishedAt == nil {
			return fmt.Errorf("%w: hosted catalog published_at is required for maximum age validation", ErrInvalidCatalog)
		}
		if currentTime.Sub(*doc.PublishedAt) > options.MaxAge {
			return fmt.Errorf("%w: hosted catalog exceeds maximum age %s", ErrInvalidCatalog, options.MaxAge)
		}
	}
	return nil
}

func loadHostedCache(options HostedOptions) (Document, error) {
	file, err := os.Open(options.CachePath)
	if err != nil {
		return Document{}, fmt.Errorf("open hosted catalog cache %s: %w", options.CachePath, err)
	}
	defer func() { _ = file.Close() }()
	data, err := readHostedCatalogBytes(file, options.MaxBytes)
	if err != nil {
		return Document{}, err
	}
	return parseHostedCatalog(data, options)
}

func writeHostedCache(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create hosted catalog cache directory %s: %w", dir, err)
	}
	file, err := os.CreateTemp(dir, ".bootup-catalog-cache-*")
	if err != nil {
		return fmt.Errorf("create hosted catalog cache temp file: %w", err)
	}
	tempPath := file.Name()
	defer func() { _ = os.Remove(tempPath) }()
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return fmt.Errorf("write hosted catalog cache temp file: %w", err)
	}
	if err := file.Chmod(0o644); err != nil {
		_ = file.Close()
		return fmt.Errorf("chmod hosted catalog cache temp file: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close hosted catalog cache temp file: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("replace hosted catalog cache %s: %w", path, err)
	}
	return nil
}

func verifySHA256(data []byte, wantHex string) error {
	want, err := hex.DecodeString(wantHex)
	if err != nil {
		return fmt.Errorf("%w: hosted catalog SHA-256 is not valid hex: %w", ErrInvalidCatalog, err)
	}
	if len(want) != sha256.Size {
		return fmt.Errorf("%w: hosted catalog SHA-256 length is %d bytes, want %d", ErrInvalidCatalog, len(want), sha256.Size)
	}
	got := sha256.Sum256(data)
	if subtle.ConstantTimeCompare(got[:], want) != 1 {
		return fmt.Errorf("%w: hosted catalog SHA-256 mismatch", ErrInvalidCatalog)
	}
	return nil
}

func verifyEd25519(data []byte, signature []byte, publicKey []byte) error {
	if len(signature) != ed25519.SignatureSize {
		return fmt.Errorf("%w: hosted catalog Ed25519 signature length is %d bytes, want %d", ErrInvalidCatalog, len(signature), ed25519.SignatureSize)
	}
	if len(publicKey) != ed25519.PublicKeySize {
		return fmt.Errorf("%w: hosted catalog Ed25519 public key length is %d bytes, want %d", ErrInvalidCatalog, len(publicKey), ed25519.PublicKeySize)
	}
	if !ed25519.Verify(publicKey, data, signature) {
		return fmt.Errorf("%w: hosted catalog Ed25519 signature verification failed", ErrInvalidCatalog)
	}
	return nil
}
