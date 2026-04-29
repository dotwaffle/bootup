package kernelconfig_test

import (
	"os/exec"
	"strings"
	"testing"
)

func TestCheckKernelConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		fixture    string
		wantOK     bool
		wantStderr string
	}{
		{
			name:    "pass",
			fixture: "testdata/pass.config",
			wantOK:  true,
		},
		{
			name:       "missing",
			fixture:    "testdata/missing.config",
			wantStderr: "CONFIG_IP_PNP is not set",
		},
		{
			name:       "modular",
			fixture:    "testdata/modular.config",
			wantStderr: "CONFIG_E1000=m, want CONFIG_E1000=y",
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
			if tt.wantStderr != "" && !strings.Contains(string(output), tt.wantStderr) {
				t.Fatalf("output = %q, want substring %q", output, tt.wantStderr)
			}
		})
	}
}
