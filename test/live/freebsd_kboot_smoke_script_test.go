package live_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestFreeBSDKbootSmokeScriptPrintsArtifactURLs(t *testing.T) {
	t.Parallel()

	script := filepath.Join(repoRoot(t), "scripts", "smoke-freebsd-kboot.sh")
	command := exec.Command("bash", script, "--print-artifact-urls")
	command.Env = append(os.Environ(),
		"BOOTUP_FREEBSD_VERSION=15.0-RELEASE",
		"BOOTUP_FREEBSD_ARCH=amd64",
	)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("print artifact URLs: %v\n%s", err, output)
	}

	got := string(output)
	for _, want := range []string{
		"base.txz https://download.freebsd.org/releases/amd64/amd64/15.0-RELEASE/base.txz",
		"bootonly.iso.xz https://download.freebsd.org/releases/amd64/amd64/ISO-IMAGES/15.0/FreeBSD-15.0-RELEASE-amd64-bootonly.iso.xz",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("artifact URL output = %q, want %q", got, want)
		}
	}
}
