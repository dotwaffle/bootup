package providerconfig_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/dotwaffle/bootup/internal/providerconfig"
)

func TestLoadFileAppliesProviderEntries(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	keyringPath, keyring := writeKeyring(t, dir)
	configPath := filepath.Join(dir, "providers.json")
	writeFile(t, configPath, []byte(`{
		"providers": {
			"debian": {
				"mirror_url": "https://mirror.example/debian",
				"keyring_path": `+quote(keyringPath)+`
			},
			"ubuntu": {
				"release_url": "https://releases.example/26.04",
				"keyring_path": `+quote(keyringPath)+`,
				"kernel_sha256": "`+strings.Repeat("a", 64)+`",
				"initrd_sha256": "`+strings.Repeat("b", 64)+`"
			}
		}
	}`))

	config, err := providerconfig.LoadFile(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if config.Debian.MirrorURL != "https://mirror.example/debian" {
		t.Fatalf("Debian mirror URL = %q", config.Debian.MirrorURL)
	}
	if !bytes.Equal(config.Debian.Keyring, keyring) {
		t.Fatal("Debian keyring does not match configured file")
	}
	if config.Ubuntu.ReleaseURL != "https://releases.example/26.04" {
		t.Fatalf("Ubuntu release URL = %q", config.Ubuntu.ReleaseURL)
	}
	if !bytes.Equal(config.Ubuntu.Keyring, keyring) {
		t.Fatal("Ubuntu keyring does not match configured file")
	}
	if config.Ubuntu.KernelSHA256 != strings.Repeat("a", 64) {
		t.Fatalf("Ubuntu kernel sha256 = %q", config.Ubuntu.KernelSHA256)
	}
	if config.Ubuntu.InitrdSHA256 != strings.Repeat("b", 64) {
		t.Fatalf("Ubuntu initrd sha256 = %q", config.Ubuntu.InitrdSHA256)
	}
}

func TestLoadFileRejectsInvalidConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	keyringPath, _ := writeKeyring(t, dir)
	missingPath := filepath.Join(dir, "missing.gpg")

	tests := []struct {
		name string
		json string
	}{
		{
			name: "malformed json",
			json: `{"providers":`,
		},
		{
			name: "unknown provider",
			json: `{"providers":{"fedora":{}}}`,
		},
		{
			name: "invalid hash",
			json: `{"providers":{"ubuntu":{"kernel_sha256":"abc","initrd_sha256":"` + strings.Repeat("b", 64) + `"}}}`,
		},
		{
			name: "partial hash pins",
			json: `{"providers":{"ubuntu":{"kernel_sha256":"` + strings.Repeat("a", 64) + `"}}}`,
		},
		{
			name: "unreadable keyring",
			json: `{"providers":{"debian":{"keyring_path":` + quote(missingPath) + `}}}`,
		},
		{
			name: "unknown provider field",
			json: `{"providers":{"debian":{"keyring_path":` + quote(keyringPath) + `,"release_url":"https://example.invalid"}}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			configPath := filepath.Join(t.TempDir(), "providers.json")
			writeFile(t, configPath, []byte(tt.json))
			if _, err := providerconfig.LoadFile(configPath); err == nil {
				t.Fatal("load config succeeded, want error")
			}
		})
	}
}

func writeKeyring(t *testing.T, dir string) (string, []byte) {
	t.Helper()

	entity, err := openpgp.NewEntity("Provider Trust", "", "trust@example.test", nil)
	if err != nil {
		t.Fatalf("new entity: %v", err)
	}
	var keyring bytes.Buffer
	armorWriter, err := armor.Encode(&keyring, openpgp.PublicKeyType, nil)
	if err != nil {
		t.Fatalf("armor keyring: %v", err)
	}
	if err := entity.Serialize(armorWriter); err != nil {
		t.Fatalf("serialize keyring: %v", err)
	}
	if err := armorWriter.Close(); err != nil {
		t.Fatalf("close armor: %v", err)
	}

	path := filepath.Join(dir, "provider.gpg")
	writeFile(t, path, keyring.Bytes())
	return path, keyring.Bytes()
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()

	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func quote(value string) string {
	return `"` + value + `"`
}
