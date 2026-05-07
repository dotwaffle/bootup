package secrets_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/dotwaffle/bootup/internal/secrets"
)

func TestLoadValidatesFileBackedSecretInputs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	secretPath := filepath.Join(dir, "installer-password")
	if err := os.WriteFile(secretPath, []byte("correct horse battery staple"), 0o600); err != nil {
		t.Fatalf("write secret: %v", err)
	}

	store, err := secrets.Load([]secrets.Selection{{
		ID:   "installer-password",
		Path: secretPath,
	}}, secrets.Options{})
	if err != nil {
		t.Fatalf("load secrets: %v", err)
	}
	if !store.Has("installer-password") {
		t.Fatal("store does not contain installer-password")
	}
	if got := store.IDs(); !slices.Equal(got, []string{"installer-password"}) {
		t.Fatalf("secret IDs = %#v, want installer-password", got)
	}
}

func TestLoadRejectsUnsafeSecretInputs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	readablePath := filepath.Join(dir, "readable")
	if err := os.WriteFile(readablePath, []byte("secret"), 0o644); err != nil {
		t.Fatalf("write readable secret: %v", err)
	}
	largePath := filepath.Join(dir, "large")
	if err := os.WriteFile(largePath, []byte(strings.Repeat("x", 5)), 0o600); err != nil {
		t.Fatalf("write large secret: %v", err)
	}

	tests := []struct {
		name      string
		selector  secrets.Selection
		options   secrets.Options
		wantError string
	}{
		{
			name:      "empty id",
			selector:  secrets.Selection{Path: readablePath},
			wantError: "secret input ID is required",
		},
		{
			name:      "relative path",
			selector:  secrets.Selection{ID: "installer-password", Path: "secret.txt"},
			wantError: "absolute path",
		},
		{
			name:      "missing file",
			selector:  secrets.Selection{ID: "installer-password", Path: filepath.Join(dir, "missing")},
			wantError: "read secret input installer-password",
		},
		{
			name:      "directory",
			selector:  secrets.Selection{ID: "installer-password", Path: dir},
			wantError: "regular file",
		},
		{
			name:      "group or other readable",
			selector:  secrets.Selection{ID: "installer-password", Path: readablePath},
			wantError: "group or other readable",
		},
		{
			name:      "too large",
			selector:  secrets.Selection{ID: "installer-password", Path: largePath},
			options:   secrets.Options{MaxBytes: 4},
			wantError: "exceeds maximum size",
		},
		{
			name:      "duplicate id",
			selector:  secrets.Selection{ID: "installer-password", Path: readablePath},
			wantError: "duplicate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			selections := []secrets.Selection{tt.selector}
			if tt.name == "duplicate id" {
				selections = append(selections, tt.selector)
			}
			_, err := secrets.Load(selections, tt.options)
			if err == nil {
				t.Fatal("load secrets succeeded, want error")
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("load error = %q, want %q", err, tt.wantError)
			}
		})
	}
}

func TestLoadDoesNotExposeSecretInputPathsInErrors(t *testing.T) {
	t.Parallel()

	secretPath := filepath.Join(t.TempDir(), "missing-secret")
	_, err := secrets.Load([]secrets.Selection{{
		ID:   "installer-password",
		Path: secretPath,
	}}, secrets.Options{})
	if err == nil {
		t.Fatal("load secrets succeeded, want error")
	}
	if strings.Contains(err.Error(), secretPath) {
		t.Fatalf("load error = %q, want redacted path", err)
	}
}

func TestStoreStagesSecretFilePrivately(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	secretPath := filepath.Join(dir, "installer-password")
	if err := os.WriteFile(secretPath, []byte("secret value"), 0o600); err != nil {
		t.Fatalf("write secret: %v", err)
	}
	store, err := secrets.Load([]secrets.Selection{{ID: "installer-password", Path: secretPath}}, secrets.Options{})
	if err != nil {
		t.Fatalf("load secrets: %v", err)
	}

	staged, err := store.StageFile("installer-password", t.TempDir(), "installer-password")
	if err != nil {
		t.Fatalf("stage secret: %v", err)
	}
	data, err := os.ReadFile(staged)
	if err != nil {
		t.Fatalf("read staged secret: %v", err)
	}
	if string(data) != "secret value" {
		t.Fatalf("staged secret = %q, want original bytes", data)
	}
	info, err := os.Stat(staged)
	if err != nil {
		t.Fatalf("stat staged secret: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("staged mode = %03o, want 600", info.Mode().Perm())
	}
}
