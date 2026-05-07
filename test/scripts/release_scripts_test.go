package scripts_test

import (
	"os"
	"strings"
	"testing"
)

func TestBuildReleaseScriptStampsBootupBuildInfo(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("../../scripts/build-release.sh")
	if err != nil {
		t.Fatalf("read build-release.sh: %v", err)
	}
	script := string(data)
	for _, want := range []string{
		`build_date="$(release_build_date)"`,
		`dirty_state="$(release_dirty_state)"`,
		`-X github.com/dotwaffle/bootup/internal/buildinfo.version=${release_version}`,
		`-X github.com/dotwaffle/bootup/internal/buildinfo.commit=${commit}`,
		`-X github.com/dotwaffle/bootup/internal/buildinfo.date=${build_date}`,
		`-X github.com/dotwaffle/bootup/internal/buildinfo.dirty=${dirty_state}`,
		`bootupBuild: {`,
		`version: $bootupBuildVersion`,
		`gitCommit: $bootupBuildCommit`,
		`buildDate: $bootupBuildDate`,
		`dirty: $bootupBuildDirty`,
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("build-release.sh is missing %q", want)
		}
	}
}

func TestCheckReleaseArtifactsScriptValidatesBootupBuildInfo(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("../../scripts/check-release-artifacts.sh")
	if err != nil {
		t.Fatalf("read check-release-artifacts.sh: %v", err)
	}
	script := string(data)
	for _, want := range []string{
		`bootup_build_version="$(json_string "${manifest_path}" '.bootupBuild.version')"`,
		`bootup_version_line`,
		`"${release_dir}/${bootup_binary_name}" --version`,
		`binary build version = %q, manifest says %q`,
		`binary build commit = %q, manifest says %q`,
		`binary build date = %q, manifest says %q`,
		`binary dirty state = %q, manifest says %q`,
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("check-release-artifacts.sh is missing %q", want)
		}
	}
}
