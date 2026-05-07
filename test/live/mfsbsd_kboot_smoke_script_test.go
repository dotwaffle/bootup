package live_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMFSBSDProductSmokeScriptDryRunShowsBootFlow(t *testing.T) {
	t.Parallel()

	script := filepath.Join(repoRoot(t), "scripts", "smoke-mfsbsd-kboot-target.sh")
	command := exec.Command("bash", script, "--dry-run")
	command.Env = append(os.Environ(), "BOOTUP_LIVE_MFSBSD_KBOOT_SMOKE=1")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("dry run script: %v\n%s", err, output)
	}
	got := string(output)
	for _, want := range []string{
		"bootup --mode=boot-target --target=mfsbsd-142-amd64 --staging-dir=/tmp/bootup --prepare-runtime",
		"--net-iface=eth0",
		"--net-dns=10.0.2.3",
		"scripts/build-initramfs.sh",
		"scripts/build-iso.sh",
		"scripts/run-qemu-iso.sh",
		"target marker: login:",
		"md0: Preloaded",
		"Trying to mount root from ufs:/dev/md0",
		"FreeBSD/amd64 (mfsbsd)",
		"login:",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("dry run output = %q, want %q", got, want)
		}
	}
}

func TestMFSBSDProductSmokeScriptRequiresOptIn(t *testing.T) {
	t.Parallel()

	script := filepath.Join(repoRoot(t), "scripts", "smoke-mfsbsd-kboot-target.sh")
	command := exec.Command("bash", script, "--dry-run")
	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("script succeeded without opt-in: %s", output)
	}
	if !strings.Contains(string(output), "BOOTUP_LIVE_MFSBSD_KBOOT_SMOKE=1 is required") {
		t.Fatalf("script output = %q, want opt-in error", output)
	}
}

func TestCatalogLiveSmokeCanSelectMFSBSDTarget(t *testing.T) {
	t.Parallel()

	target := catalogSmokeTarget(t, "mfsbsd-142-amd64")
	if target.ProviderID != "mfsbsd" || target.Action != "freebsd-kboot" {
		t.Fatalf("target = %#v, want mfsbsd freebsd-kboot target", target)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()

	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("find repo root: %v", err)
	}
	return root
}
