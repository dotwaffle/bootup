package kernelconfig_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckKernelConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		fixture       string
		wantOK        bool
		wantStderrSub []string
	}{
		{
			name:    "pass",
			fixture: "testdata/pass.config",
			wantOK:  true,
		},
		{
			name:    "missing",
			fixture: "testdata/missing.config",
			wantStderrSub: []string{
				"CONFIG_KALLSYMS_ALL is not set",
				"CONFIG_IP_PNP is not set",
			},
		},
		{
			name:    "modular",
			fixture: "testdata/modular.config",
			wantStderrSub: []string{
				"CONFIG_PROC_KCORE=m, want CONFIG_PROC_KCORE=y",
				"CONFIG_E1000=m, want CONFIG_E1000=y",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd := exec.Command("bash", "../../scripts/check-kernel-config.sh", tt.fixture)
			output, err := cmd.CombinedOutput()
			if tt.wantOK && err != nil {
				t.Fatalf("check config: %v\n%s", err, output)
			}
			if !tt.wantOK && err == nil {
				t.Fatalf("check config succeeded, want failure\n%s", output)
			}
			for _, want := range tt.wantStderrSub {
				if !strings.Contains(string(output), want) {
					t.Fatalf("output = %q, want substring %q", output, want)
				}
			}
		})
	}
}

func TestCheckKernelConfigCanSkipISOMountRequirements(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("testdata/pass.config")
	if err != nil {
		t.Fatalf("read pass fixture: %v", err)
	}
	data = bytes.ReplaceAll(data, []byte("CONFIG_ISO9660_FS=y"), []byte("# CONFIG_ISO9660_FS is not set"))
	data = bytes.ReplaceAll(data, []byte("CONFIG_BLK_DEV_LOOP=y"), []byte("# CONFIG_BLK_DEV_LOOP is not set"))
	config := filepath.Join(t.TempDir(), "no-iso-mount.config")
	if err := os.WriteFile(config, data, 0o644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	cmd := exec.Command("bash", "../../scripts/check-kernel-config.sh", config)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("check config succeeded, want ISO mount requirement failure\n%s", output)
	}
	for _, want := range []string{
		"CONFIG_ISO9660_FS is not set",
		"CONFIG_BLK_DEV_LOOP is not set",
	} {
		if !strings.Contains(string(output), want) {
			t.Fatalf("output = %q, want substring %q", output, want)
		}
	}

	cmd = exec.Command("bash", "../../scripts/check-kernel-config.sh", config)
	cmd.Env = append(os.Environ(), "BOOTUP_KERNEL_CONFIG_REQUIRE_ISO_MOUNT=0")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("check config with ISO mount disabled: %v\n%s", err, output)
	}
}
