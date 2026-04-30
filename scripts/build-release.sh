#!/usr/bin/env bash
set -euo pipefail

usage() {
	cat >&2 <<'USAGE'
usage: scripts/build-release.sh [output-dir]

Environment:
  BOOTUP_RELEASE_VERSION       release version, default tag or dev-<commit>
  BOOTUP_RELEASE_ARCH          target architecture, default amd64
  BOOTUP_RELEASE_KERNEL        existing kernel image to publish
  BOOTUP_RELEASE_KERNEL_CONFIG existing kernel config to publish
  BOOTUP_RELEASE_REBUILD_KERNEL=1
                               rebuild kernel even if dist/kernel has one
  BOOTUP_KERNEL_VERSION        kernel version passed to build-kernel.sh
  BOOTUP_RELEASE_WORK          working directory, default dist/release-work

The output directory is refreshed by removing existing bootup-* files before
writing the new release set.
USAGE
}

require_cmd() {
	if ! command -v "$1" >/dev/null 2>&1; then
		printf '%s not found; install %s to build a bootup release\n' "$1" "$1" >&2
		exit 1
	fi
}

default_release_version() {
	if git describe --tags --exact-match >/dev/null 2>&1; then
		git describe --tags --exact-match
		return
	fi
	printf 'dev-%s\n' "$(git rev-parse --short HEAD)"
}

validate_release_version() {
	local version="$1"

	if [[ ! "${version}" =~ ^[A-Za-z0-9._+-]+$ ]]; then
		printf 'bad release version %q; use only letters, digits, dot, underscore, plus, or hyphen\n' "${version}" >&2
		exit 1
	fi
}

latest_kernel() {
	local kernel_dir="$1"
	local -a kernels=()

	if [[ -d "${kernel_dir}" ]]; then
		mapfile -t kernels < <(find "${kernel_dir}" -maxdepth 1 -type f -name 'linux-*-bootup-amd64-bzImage' | sort -V)
	fi
	if [[ "${#kernels[@]}" -eq 0 ]]; then
		return 1
	fi
	printf '%s\n' "${kernels[-1]}"
}

kernel_config_for() {
	local kernel="$1"
	printf '%s.config\n' "${kernel%-bzImage}"
}

kernel_version_from_path() {
	local path="$1"
	local base

	base="$(basename -- "${path}")"
	if [[ "${base}" =~ linux-([0-9]+[.][0-9]+([.][0-9]+)?)-bootup-amd64 ]]; then
		printf '%s\n' "${BASH_REMATCH[1]}"
		return
	fi
	printf 'cannot determine kernel version from %s\n' "${base}" >&2
	exit 1
}

sha256_file() {
	sha256sum "$1" | awk '{print $1}'
}

artifact_entry() {
	local role="$1"
	local name="$2"
	local path="$3"
	local bytes
	local sha256

	bytes="$(stat -c '%s' "${path}")"
	sha256="$(sha256_file "${path}")"
	jq --compact-output --null-input \
		--arg role "${role}" \
		--arg name "${name}" \
		--arg path "${name}" \
		--arg sha256 "${sha256}" \
		--argjson bytes "${bytes}" \
		'{role: $role, name: $name, path: $path, bytes: $bytes, sha256: $sha256}'
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
	usage
	exit 0
fi

require_cmd awk
require_cmd find
require_cmd git
require_cmd go
require_cmd install
require_cmd jq
require_cmd sha256sum
require_cmd stat

repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
arch="${BOOTUP_RELEASE_ARCH:-amd64}"
if [[ "${arch}" != "amd64" ]]; then
	printf 'unsupported release architecture %q; only amd64 is currently supported\n' "${arch}" >&2
	exit 1
fi

release_version="${BOOTUP_RELEASE_VERSION:-$(default_release_version)}"
validate_release_version "${release_version}"

out_dir="${1:-${BOOTUP_RELEASE_OUT:-${repo_root}/dist/release}}"
work_dir="${BOOTUP_RELEASE_WORK:-${repo_root}/dist/release-work}"
kernel_dir="${BOOTUP_RELEASE_KERNEL_DIR:-${repo_root}/dist/kernel}"
commit="$(git rev-parse HEAD)"

mkdir -p "${out_dir}" "${work_dir}" "${kernel_dir}"
out_dir="$(cd -- "${out_dir}" && pwd)"
work_dir="$(cd -- "${work_dir}" && pwd)"
kernel_dir="$(cd -- "${kernel_dir}" && pwd)"
find "${out_dir}" -maxdepth 1 -type f -name 'bootup-*' -delete

binary_name="bootup-${release_version}-linux-${arch}"
binary_path="${out_dir}/${binary_name}"
GOOS=linux GOARCH=amd64 GOAMD64="${GOAMD64:-v1}" \
	go build -trimpath -o "${binary_path}" ./cmd/bootup

if [[ -n "${BOOTUP_RELEASE_KERNEL:-}" ]]; then
	kernel_src="${BOOTUP_RELEASE_KERNEL}"
	kernel_config_src="${BOOTUP_RELEASE_KERNEL_CONFIG:-$(kernel_config_for "${kernel_src}")}"
elif [[ "${BOOTUP_RELEASE_REBUILD_KERNEL:-}" == "1" ]] || ! kernel_src="$(latest_kernel "${kernel_dir}")"; then
	if [[ -n "${BOOTUP_KERNEL_VERSION:-}" ]]; then
		BOOTUP_KERNEL_VERSION="${BOOTUP_KERNEL_VERSION}" "${repo_root}/scripts/build-kernel.sh" "${kernel_dir}"
	else
		"${repo_root}/scripts/build-kernel.sh" "${kernel_dir}"
	fi
	kernel_src="$(latest_kernel "${kernel_dir}")"
	kernel_config_src="$(kernel_config_for "${kernel_src}")"
else
	kernel_config_src="$(kernel_config_for "${kernel_src}")"
fi
if [[ ! -r "${kernel_src}" ]]; then
	printf 'kernel is not readable: %s\n' "${kernel_src}" >&2
	exit 1
fi
if [[ ! -r "${kernel_config_src}" ]]; then
	printf 'kernel config is not readable: %s\n' "${kernel_config_src}" >&2
	exit 1
fi

kernel_version="$(kernel_version_from_path "${kernel_src}")"
kernel_name="bootup-${release_version}-linux-${kernel_version}-${arch}-bzImage"
kernel_config_name="bootup-${release_version}-linux-${kernel_version}-${arch}.config"
kernel_path="${out_dir}/${kernel_name}"
kernel_config_path="${out_dir}/${kernel_config_name}"
install -m 0644 "${kernel_src}" "${kernel_path}"
install -m 0644 "${kernel_config_src}" "${kernel_config_path}"

initramfs_raw="${work_dir}/bootup-${release_version}-initramfs-${arch}.cpio"
initramfs_name="bootup-${release_version}-initramfs-${arch}.cpio.zst"
initramfs_path="${out_dir}/${initramfs_name}"
BOOTUP_INITRAMFS_ZSTD="${initramfs_path}" \
	"${repo_root}/scripts/build-initramfs.sh" \
	"${initramfs_raw}" \
	'bootup --mode=menu --ui=auto --prepare-runtime' \
	'' \
	''

iso_name="bootup-${release_version}-hybrid-${arch}.iso"
iso_path="${out_dir}/${iso_name}"
BOOTUP_ISO_KERNEL="${kernel_path}" \
	BOOTUP_ISO_INITRAMFS="${initramfs_path}" \
	"${repo_root}/scripts/build-iso.sh" "${iso_path}"

manifest_name="bootup-${release_version}-${arch}-manifest.json"
manifest_path="${out_dir}/${manifest_name}"
sums_name="bootup-${release_version}-${arch}-SHA256SUMS"
sums_path="${out_dir}/${sums_name}"

artifact_entries=(
	"$(artifact_entry bootup-binary "${binary_name}" "${binary_path}")"
	"$(artifact_entry kernel-image "${kernel_name}" "${kernel_path}")"
	"$(artifact_entry kernel-config "${kernel_config_name}" "${kernel_config_path}")"
	"$(artifact_entry initramfs "${initramfs_name}" "${initramfs_path}")"
	"$(artifact_entry iso "${iso_name}" "${iso_path}")"
)

printf '%s\n' "${artifact_entries[@]}" | jq --slurp \
	--arg releaseVersion "${release_version}" \
	--arg gitCommit "${commit}" \
	--arg architecture "${arch}" \
	--arg kernelVersion "${kernel_version}" \
	'{
		schemaVersion: 1,
		releaseVersion: $releaseVersion,
		gitCommit: $gitCommit,
		architecture: $architecture,
		kernelVersion: $kernelVersion,
		trustMaterial: {
			distributionSpecificEmbedded: false,
			posture: "default release artifacts embed no distribution-specific archive keyrings or trust bundles; providers use operator-supplied trust material"
		},
		artifacts: .
	}' >"${manifest_path}"

(
	cd "${out_dir}"
	sha256sum \
		"${binary_name}" \
		"${kernel_name}" \
		"${kernel_config_name}" \
		"${initramfs_name}" \
		"${iso_name}" \
		"${manifest_name}" \
		>"${sums_path}"
)

printf 'release %s\n' "${out_dir}"
printf 'manifest %s\n' "${manifest_path}"
printf 'checksums %s\n' "${sums_path}"
