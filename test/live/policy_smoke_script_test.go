package live_test

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

const policySmokeEnv = "BOOTUP_POLICY_SMOKE"

func TestPolicySmokeScriptUsesSignedPolicyDiagnostics(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("../../scripts/smoke-policy-target.sh")
	if err != nil {
		t.Fatalf("read policy smoke script: %v", err)
	}
	script := string(data)
	for _, want := range []string{
		"bootup-policy-sign",
		"--mode=policy-target",
		"--diagnostics-dir",
		"opensuse-leap-160-amd64-netboot",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("policy smoke script is missing %q", want)
		}
	}
}

func TestPolicySmokeScriptRuns(t *testing.T) {
	if os.Getenv(policySmokeEnv) != "1" {
		t.Skip(policySmokeEnv + "=1 is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)

	command := exec.CommandContext(ctx, "../../scripts/smoke-policy-target.sh")
	command.Env = append(os.Environ(), policySmokeEnv+"=1")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("run policy smoke script: %v\n%s", err, output)
	}
}
