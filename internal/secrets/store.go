// Package secrets loads local file-backed secret inputs for providers.
package secrets

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dotwaffle/bootup/internal/provider"
)

const defaultMaxBytes int64 = 1 << 20

// Selection identifies one operator-supplied secret input file.
type Selection struct {
	ID   string
	Path string
}

// Options controls secret input validation.
type Options struct {
	MaxBytes int64
}

// Store contains validated secret input bytes.
type Store struct {
	entries map[string][]byte
}

// Load validates and reads file-backed secret inputs.
func Load(selections []Selection, options Options) (*Store, error) {
	maxBytes := options.MaxBytes
	if maxBytes <= 0 {
		maxBytes = defaultMaxBytes
	}
	seen := make(map[string]struct{}, len(selections))
	for _, selection := range selections {
		id := strings.TrimSpace(selection.ID)
		if id == "" {
			return nil, fmt.Errorf("%w: secret input ID is required", provider.ErrInvalidSecretInput)
		}
		if _, ok := seen[id]; ok {
			return nil, fmt.Errorf("%w: duplicate secret input %q", provider.ErrInvalidSecretInput, id)
		}
		seen[id] = struct{}{}
	}
	entries := make(map[string][]byte, len(selections))
	for _, selection := range selections {
		id := strings.TrimSpace(selection.ID)
		if strings.TrimSpace(selection.Path) != selection.Path {
			return nil, fmt.Errorf("%w: secret input %s path has surrounding whitespace", provider.ErrInvalidSecretInput, id)
		}
		if !filepath.IsAbs(selection.Path) {
			return nil, fmt.Errorf("%w: secret input %s path must be an absolute path", provider.ErrInvalidSecretInput, id)
		}
		data, err := readSecretFile(id, selection.Path, maxBytes)
		if err != nil {
			return nil, err
		}
		entries[id] = data
	}
	return &Store{entries: entries}, nil
}

func readSecretFile(id string, path string, maxBytes int64) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("%w: read secret input %s: %s", provider.ErrInvalidSecretInput, id, secretFileErrorReason(err))
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: secret input %s must be a regular file", provider.ErrInvalidSecretInput, id)
	}
	if info.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("%w: secret input %s must not be group or other readable", provider.ErrInvalidSecretInput, id)
	}
	if info.Size() > maxBytes {
		return nil, fmt.Errorf("%w: secret input %s exceeds maximum size %d", provider.ErrInvalidSecretInput, id, maxBytes)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: read secret input %s: %s", provider.ErrInvalidSecretInput, id, secretFileErrorReason(err))
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("%w: secret input %s exceeds maximum size %d", provider.ErrInvalidSecretInput, id, maxBytes)
	}
	return data, nil
}

// IDs returns sorted secret input IDs.
func (s *Store) IDs() []string {
	if s == nil {
		return nil
	}
	ids := make([]string, 0, len(s.entries))
	for id := range s.entries {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// Has reports whether id is present.
func (s *Store) Has(id string) bool {
	if s == nil {
		return false
	}
	_, ok := s.entries[id]
	return ok
}

// StageFile writes a private copy of id under dir using name.
func (s *Store) StageFile(id string, dir string, name string) (string, error) {
	if s == nil {
		return "", fmt.Errorf("%w: secret input %s is missing", provider.ErrInvalidSecretInput, id)
	}
	data, ok := s.entries[id]
	if !ok {
		return "", fmt.Errorf("%w: secret input %s is missing", provider.ErrInvalidSecretInput, id)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = id
	}
	if filepath.Base(name) != name || strings.ContainsAny(name, `/\`) {
		return "", fmt.Errorf("%w: secret stage name %q must be a filename", provider.ErrInvalidSecretInput, name)
	}
	if strings.TrimSpace(dir) == "" {
		return "", errors.New("secret staging dir is required")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create secret staging dir: %s", secretFileErrorReason(err))
	}
	targetPath := filepath.Join(dir, name)
	temp, err := os.CreateTemp(dir, "."+name+".")
	if err != nil {
		return "", fmt.Errorf("create secret temp file: %s", secretFileErrorReason(err))
	}
	tempPath := temp.Name()
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		_ = os.Remove(tempPath)
		return "", fmt.Errorf("write secret temp file: %s", secretFileErrorReason(err))
	}
	if err := temp.Close(); err != nil {
		_ = os.Remove(tempPath)
		return "", fmt.Errorf("close secret temp file: %s", secretFileErrorReason(err))
	}
	if err := os.Chmod(tempPath, 0o600); err != nil {
		_ = os.Remove(tempPath)
		return "", fmt.Errorf("chmod secret temp file: %s", secretFileErrorReason(err))
	}
	if err := os.Rename(tempPath, targetPath); err != nil {
		_ = os.Remove(tempPath)
		return "", fmt.Errorf("stage secret file: %s", secretFileErrorReason(err))
	}
	return targetPath, nil
}

func secretFileErrorReason(err error) string {
	switch {
	case errors.Is(err, os.ErrNotExist):
		return "file does not exist"
	case errors.Is(err, os.ErrPermission):
		return "permission denied"
	default:
		return "filesystem error"
	}
}
