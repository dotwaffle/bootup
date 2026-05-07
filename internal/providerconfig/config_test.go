package providerconfig_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/dotwaffle/bootup/internal/provider"
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
				"discovery_url": "https://discovery.example/debian",
				"discovery_timeout": "750ms",
				"keyring_path": `+quote(keyringPath)+`,
				"lifecycle": {
					"trixie": {
						"status": "supported",
						"source": "operator",
						"date": "2028-06-30"
					}
				}
			},
			"ubuntu": {
				"release_url": "https://releases.example/26.04",
				"discovery_url": "https://releases.example/releases/",
				"discovery_timeout": "2s",
				"keyring_path": `+quote(keyringPath)+`,
				"kernel_sha256": "`+strings.Repeat("a", 64)+`",
				"initrd_sha256": "`+strings.Repeat("b", 64)+`",
				"lifecycle": {
					"26.04": {
						"status": "supported",
						"source": "operator"
					}
				}
			},
			"fedora": {
				"release_url": "https://download.example/fedora/releases/44/Server/x86_64/os",
				"discovery_url": "https://download.example/fedora/releases/",
				"discovery_timeout": "3s",
				"kernel_sha256": "`+strings.Repeat("c", 64)+`",
				"initrd_sha256": "`+strings.Repeat("d", 64)+`"
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
	if config.Debian.DiscoveryURL != "https://discovery.example/debian" {
		t.Fatalf("Debian discovery URL = %q", config.Debian.DiscoveryURL)
	}
	if config.Debian.DiscoveryTimeout != 750*time.Millisecond {
		t.Fatalf("Debian discovery timeout = %s, want 750ms", config.Debian.DiscoveryTimeout)
	}
	if got := config.Debian.Lifecycle["trixie"]; got.Status != provider.LifecycleSupported || got.Source != "operator" || got.Date != "2028-06-30" {
		t.Fatalf("Debian trixie lifecycle = %#v", got)
	}
	if !bytes.Equal(config.Debian.Keyring, keyring) {
		t.Fatal("Debian keyring does not match configured file")
	}
	if config.Ubuntu.ReleaseURL != "https://releases.example/26.04" {
		t.Fatalf("Ubuntu release URL = %q", config.Ubuntu.ReleaseURL)
	}
	if config.Ubuntu.DiscoveryURL != "https://releases.example/releases" {
		t.Fatalf("Ubuntu discovery URL = %q", config.Ubuntu.DiscoveryURL)
	}
	if config.Ubuntu.DiscoveryTimeout != 2*time.Second {
		t.Fatalf("Ubuntu discovery timeout = %s, want 2s", config.Ubuntu.DiscoveryTimeout)
	}
	if got := config.Ubuntu.Lifecycle["26.04"]; got.Status != provider.LifecycleSupported || got.Source != "operator" {
		t.Fatalf("Ubuntu 26.04 lifecycle = %#v", got)
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
	if config.Fedora.ReleaseURL != "https://download.example/fedora/releases/44/Server/x86_64/os" {
		t.Fatalf("Fedora release URL = %q", config.Fedora.ReleaseURL)
	}
	if config.Fedora.DiscoveryURL != "https://download.example/fedora/releases" {
		t.Fatalf("Fedora discovery URL = %q", config.Fedora.DiscoveryURL)
	}
	if config.Fedora.DiscoveryTimeout != 3*time.Second {
		t.Fatalf("Fedora discovery timeout = %s, want 3s", config.Fedora.DiscoveryTimeout)
	}
	if config.Fedora.KernelSHA256 != strings.Repeat("c", 64) {
		t.Fatalf("Fedora kernel sha256 = %q", config.Fedora.KernelSHA256)
	}
	if config.Fedora.InitrdSHA256 != strings.Repeat("d", 64) {
		t.Fatalf("Fedora initrd sha256 = %q", config.Fedora.InitrdSHA256)
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
			json: `{"providers":{"arch":{}}}`,
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
			name: "invalid Fedora release url",
			json: `{"providers":{"fedora":{"release_url":"file:///srv/fedora"}}}`,
		},
		{
			name: "invalid Fedora hash",
			json: `{"providers":{"fedora":{"kernel_sha256":"abc","initrd_sha256":"` + strings.Repeat("d", 64) + `"}}}`,
		},
		{
			name: "invalid Fedora discovery url",
			json: `{"providers":{"fedora":{"discovery_url":"file:///srv/fedora"}}}`,
		},
		{
			name: "invalid Fedora discovery timeout",
			json: `{"providers":{"fedora":{"discovery_timeout":"zero"}}}`,
		},
		{
			name: "unreadable keyring",
			json: `{"providers":{"debian":{"keyring_path":` + quote(missingPath) + `}}}`,
		},
		{
			name: "unknown provider field",
			json: `{"providers":{"debian":{"keyring_path":` + quote(keyringPath) + `,"release_url":"https://example.invalid"}}}`,
		},
		{
			name: "invalid discovery url",
			json: `{"providers":{"debian":{"discovery_url":"file:///srv/debian"}}}`,
		},
		{
			name: "invalid discovery timeout",
			json: `{"providers":{"ubuntu":{"discovery_timeout":"eventually"}}}`,
		},
		{
			name: "invalid lifecycle status",
			json: `{"providers":{"debian":{"lifecycle":{"trixie":{"status":"trusted","source":"operator"}}}}}`,
		},
		{
			name: "invalid lifecycle date",
			json: `{"providers":{"ubuntu":{"lifecycle":{"26.04":{"status":"supported","source":"operator","date":"soon"}}}}}`,
		},
		{
			name: "missing lifecycle source",
			json: `{"providers":{"debian":{"lifecycle":{"trixie":{"status":"supported"}}}}}`,
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
