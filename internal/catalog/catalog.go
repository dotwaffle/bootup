// Package catalog loads static provider target catalogs.
package catalog

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	_ "embed"

	"github.com/dotwaffle/bootup/internal/provider"
)

const schemaVersion = 1

//go:embed default.json
var defaultCatalog []byte

// ErrInvalidCatalog is returned when a static catalog cannot be used.
var ErrInvalidCatalog = errors.New("invalid catalog")

// Document is a versioned static catalog of concrete provider targets.
type Document struct {
	SchemaVersion int               `json:"schema_version"`
	Entries       []provider.Target `json:"targets"`
}

// LoadDefault loads bootup's embedded static catalog.
func LoadDefault(providerIDs []string) (Document, error) {
	return Parse(defaultCatalog, providerIDs)
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
