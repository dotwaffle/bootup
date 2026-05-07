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

func TestCheckReleaseArtifactsScriptValidatesGzipISOInitramfs(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("../../scripts/check-release-artifacts.sh")
	if err != nil {
		t.Fatalf("read check-release-artifacts.sh: %v", err)
	}
	script := string(data)
	if !strings.Contains(script, `require_iso_path "${iso_listing}" "/boot/bootup/initramfs.cpio.gz"`) {
		t.Fatalf("check-release-artifacts.sh does not require the gzip ISO initramfs")
	}
}

func TestBuildReleaseScriptEmbedsGzipISOInitramfs(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("../../scripts/build-release.sh")
	if err != nil {
		t.Fatalf("read build-release.sh: %v", err)
	}
	script := string(data)
	for _, want := range []string{
		`initramfs_iso_path="${work_dir}/bootup-${release_version}-iso-initramfs-${arch}.cpio.gz"`,
		`gzip -9 -c "${initramfs_raw}" >"${initramfs_iso_path}"`,
		`BOOTUP_ISO_INITRAMFS="${initramfs_iso_path}"`,
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("build-release.sh is missing %q", want)
		}
	}
}

func TestBuildISOScriptPrefersVideoConsoleForUserspace(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("../../scripts/build-iso.sh")
	if err != nil {
		t.Fatalf("read build-iso.sh: %v", err)
	}
	script := string(data)
	if !strings.Contains(script, `require_cmd gzip`) {
		t.Fatalf("build-iso.sh does not require gzip for the default ISO initramfs")
	}
	if !strings.Contains(script, `cmdline="${BOOTUP_ISO_CMDLINE:-console=tty0 panic=30 ip=::::::dhcp}"`) {
		t.Fatalf("build-iso.sh default cmdline does not leave /dev/console on tty0")
	}
	if strings.Contains(script, `BOOTUP_ISO_UINITCMD:-bootup --mode=menu --ui=auto --prepare-runtime --console-mirror=/dev/tty0`) {
		t.Fatalf("build-iso.sh default init command enables console mirror")
	}
}

func TestBuildISOScriptSupportsNonZstdInitramfsNames(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("../../scripts/build-iso.sh")
	if err != nil {
		t.Fatalf("read build-iso.sh: %v", err)
	}
	script := string(data)
	for _, want := range []string{
		`default_initramfs="${repo_root}/dist/bootup-iso-initramfs.cpio.gz"`,
		`gzip -9 -c "${raw_initramfs}" >"${initramfs}"`,
		`initramfs_default_name="$(iso_initramfs_name "${initramfs}")"`,
		`install -m 0644 "${initramfs}" "${iso_root}/boot/bootup/${initramfs_name}"`,
		`initrd /boot/bootup/${initramfs_name}`,
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("build-iso.sh is missing %q", want)
		}
	}
}
