// Package catalog loads static provider target catalogs.
package catalog

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/dotwaffle/bootup/internal/provider"
)

const schemaVersion = 1

//go:embed default.json
var defaultCatalog []byte

//go:embed source.json
var defaultSource []byte

// ErrInvalidCatalog is returned when a static catalog cannot be used.
var ErrInvalidCatalog = errors.New("invalid catalog")

// Document is a versioned static catalog of concrete provider targets.
type Document struct {
	SchemaVersion int               `json:"schema_version"`
	PublishedAt   *time.Time        `json:"published_at,omitzero"`
	ExpiresAt     *time.Time        `json:"expires_at,omitzero"`
	Entries       []provider.Target `json:"targets"`
}

// LoadDefault loads bootup's embedded static catalog.
func LoadDefault(providerIDs []string) (Document, error) {
	return Parse(defaultCatalog, providerIDs)
}

// GenerateDefault generates the embedded default static catalog from source.
func GenerateDefault() ([]byte, error) {
	return Generate(defaultSource)
}

// Generate renders a static catalog from structured source data.
func Generate(data []byte) ([]byte, error) {
	var source sourceDocument
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&source); err != nil {
		return nil, fmt.Errorf("%w: decode catalog source: %w", ErrInvalidCatalog, err)
	}
	if err := decoder.Decode(new(struct{})); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%w: decode catalog source: multiple JSON values", ErrInvalidCatalog)
	}
	if source.SchemaVersion != schemaVersion {
		return nil, fmt.Errorf("%w: unsupported source schema version %d", ErrInvalidCatalog, source.SchemaVersion)
	}

	var doc Document
	doc.SchemaVersion = schemaVersion
	for _, providerSource := range source.Providers {
		for _, targetSource := range providerSource.Targets {
			target := provider.Target{
				ID:         targetSource.ID,
				ProviderID: providerSource.ID,
				Name:       targetSource.Name,
				Action:     targetSource.Action,
				Catalog: provider.CatalogEntry{
					Distribution: targetSource.distribution(providerSource.ID),
					Release:      targetSource.Release,
					Architecture: targetSource.Architecture,
					Kind:         targetSource.Kind,
				},
				Source:    targetSource.Source,
				Lifecycle: targetSource.Lifecycle,
				Options:   targetSource.Options,
				Secrets:   targetSource.Secrets,
			}
			if err := provider.ValidateTarget(providerSource.ID, target); err != nil {
				return nil, fmt.Errorf("%w: %w", ErrInvalidCatalog, err)
			}
			doc.Entries = append(doc.Entries, target)
		}
	}
	generated, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode generated catalog: %w", err)
	}
	generated = append(generated, '\n')
	return generated, nil
}

// LoadFile loads a static catalog document from path.
func LoadFile(path string, providerIDs []string) (Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Document{}, fmt.Errorf("read catalog %s: %w", path, err)
	}
	doc, err := Parse(data, providerIDs)
	if err != nil {
		return Document{}, fmt.Errorf("parse catalog %s: %w", path, err)
	}
	return doc, nil
}

// Compose appends targets from validated catalog documents and rejects
// duplicate target IDs across sources.
func Compose(documents ...Document) (Document, error) {
	result := Document{SchemaVersion: schemaVersion}
	seenTargets := make(map[string]struct{})
	for _, doc := range documents {
		if doc.SchemaVersion != schemaVersion {
			return Document{}, fmt.Errorf("%w: unsupported schema version %d", ErrInvalidCatalog, doc.SchemaVersion)
		}
		for _, target := range doc.Entries {
			if _, ok := seenTargets[target.ID]; ok {
				return Document{}, fmt.Errorf("%w: duplicate target ID %q", ErrInvalidCatalog, target.ID)
			}
			seenTargets[target.ID] = struct{}{}
			result.Entries = append(result.Entries, target)
		}
	}
	return result, nil
}

// Parse decodes and validates a static catalog document.
func Parse(data []byte, providerIDs []string) (Document, error) {
	var doc Document
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&doc); err != nil {
		return Document{}, fmt.Errorf("%w: decode catalog: %w", ErrInvalidCatalog, err)
	}
	if err := decoder.Decode(new(struct{})); !errors.Is(err, io.EOF) {
		return Document{}, fmt.Errorf("%w: decode catalog: multiple JSON values", ErrInvalidCatalog)
	}
	if doc.SchemaVersion != schemaVersion {
		return Document{}, fmt.Errorf("%w: unsupported schema version %d", ErrInvalidCatalog, doc.SchemaVersion)
	}

	knownProviders := make(map[string]struct{}, len(providerIDs))
	for _, providerID := range providerIDs {
		knownProviders[providerID] = struct{}{}
	}
	seenTargets := make(map[string]struct{}, len(doc.Entries))
	for _, target := range doc.Entries {
		if _, ok := knownProviders[target.ProviderID]; !ok {
			return Document{}, fmt.Errorf("%w: target %s references unknown provider %q", ErrInvalidCatalog, target.ID, target.ProviderID)
		}
		if _, ok := seenTargets[target.ID]; ok {
			return Document{}, fmt.Errorf("%w: duplicate target ID %q", ErrInvalidCatalog, target.ID)
		}
		seenTargets[target.ID] = struct{}{}
		if err := provider.ValidateTarget(target.ProviderID, target); err != nil {
			return Document{}, fmt.Errorf("%w: %w", ErrInvalidCatalog, err)
		}
	}
	return doc, nil
}

// Targets returns a copy of the catalog targets for providerID.
func (d Document) Targets(providerID string) []provider.Target {
	targets := make([]provider.Target, 0)
	for _, target := range d.Entries {
		if target.ProviderID == providerID {
			targets = append(targets, target)
		}
	}
	return targets
}

type sourceDocument struct {
	SchemaVersion int              `json:"schema_version"`
	Providers     []sourceProvider `json:"providers"`
}

type sourceProvider struct {
	ID      string         `json:"id"`
	Targets []sourceTarget `json:"targets"`
}

type sourceTarget struct {
	ID           string                  `json:"id"`
	Name         string                  `json:"name"`
	Action       provider.BootAction     `json:"action,omitzero"`
	Distribution string                  `json:"distribution,omitzero"`
	Release      string                  `json:"release"`
	Architecture string                  `json:"architecture"`
	Kind         string                  `json:"kind"`
	Source       provider.SourceEntry    `json:"source,omitzero"`
	Lifecycle    provider.LifecycleEntry `json:"lifecycle,omitzero"`
	Options      []provider.TargetOption `json:"options,omitzero"`
	Secrets      []provider.SecretInput  `json:"secrets,omitzero"`
}

func (t sourceTarget) distribution(fallback string) string {
	if t.Distribution != "" {
		return t.Distribution
	}
	return fallback
}
