#!/usr/bin/env bash
set -euo pipefail

usage() {
	cat >&2 <<'USAGE'
usage: scripts/check-release-artifacts.sh [release-dir]

Environment:
  BOOTUP_RELEASE_ARCH      target architecture, default amd64
  BOOTUP_RELEASE_MANIFEST  manifest path to validate
USAGE
}

require_cmd() {
	if ! command -v "$1" >/dev/null 2>&1; then
		printf '%s not found; install %s to validate bootup release artifacts\n' "$1" "$1" >&2
		exit 1
	fi
}

find_manifest() {
	local release_dir="$1"
	local arch="$2"
	local -a manifests=()

	if [[ -n "${BOOTUP_RELEASE_MANIFEST:-}" ]]; then
		printf '%s\n' "${BOOTUP_RELEASE_MANIFEST}"
		return
	fi

	mapfile -t manifests < <(find "${release_dir}" -maxdepth 1 -type f -name "bootup-*-${arch}-manifest.json" | sort)
	if [[ "${#manifests[@]}" -ne 1 ]]; then
		printf 'expected exactly one bootup-*-%s-manifest.json in %s, found %d\n' \
			"${arch}" "${release_dir}" "${#manifests[@]}" >&2
		exit 1
	fi
	printf '%s\n' "${manifests[0]}"
}

json_string() {
	local path="$1"
	local query="$2"

	jq --raw-output "${query}" "${path}"
}

sha256_file() {
	sha256sum "$1" | awk '{print $1}'
}

require_name_in_checksums() {
	local sums="$1"
	local name="$2"

	if ! awk '{print $2}' "${sums}" | grep -Fxq "${name}"; then
		printf 'checksum file %s does not cover %s\n' "${sums}" "${name}" >&2
		exit 1
	fi
}

require_iso_path() {
	local listing="$1"
	local path="$2"

	if ! grep -Fxq "${path}" <<<"${listing}" && ! grep -Fxq "'${path}'" <<<"${listing}"; then
		printf 'ISO is missing %s\n' "${path}" >&2
		exit 1
	fi
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
	usage
	exit 0
fi

require_cmd awk
require_cmd find
require_cmd grep
require_cmd jq
require_cmd sha256sum
require_cmd stat
require_cmd xorriso

release_dir="${1:-${BOOTUP_RELEASE_OUT:-dist/release}}"
if [[ ! -d "${release_dir}" ]]; then
	printf 'release directory is not readable: %s\n' "${release_dir}" >&2
	exit 1
fi
release_dir="$(cd -- "${release_dir}" && pwd)"

arch="${BOOTUP_RELEASE_ARCH:-amd64}"
manifest_path="$(find_manifest "${release_dir}" "${arch}")"
if [[ ! -r "${manifest_path}" ]]; then
	printf 'manifest is not readable: %s\n' "${manifest_path}" >&2
	exit 1
fi
manifest_path="$(cd -- "$(dirname -- "${manifest_path}")" && pwd)/$(basename -- "${manifest_path}")"
manifest_name="$(basename -- "${manifest_path}")"

schema_version="$(json_string "${manifest_path}" '.schemaVersion')"
release_version="$(json_string "${manifest_path}" '.releaseVersion')"
manifest_arch="$(json_string "${manifest_path}" '.architecture')"
kernel_version="$(json_string "${manifest_path}" '.kernelVersion')"
trust_embedded="$(json_string "${manifest_path}" '.trustMaterial.distributionSpecificEmbedded')"

if [[ "${schema_version}" != "1" ]]; then
	printf 'manifest schemaVersion = %q, want 1\n' "${schema_version}" >&2
	exit 1
fi
if [[ -z "${release_version}" || "${release_version}" == "null" ]]; then
	printf 'manifest releaseVersion is empty\n' >&2
	exit 1
fi
if [[ "${manifest_arch}" != "${arch}" ]]; then
	printf 'manifest architecture = %q, want %q\n' "${manifest_arch}" "${arch}" >&2
	exit 1
fi
if [[ -z "${kernel_version}" || "${kernel_version}" == "null" ]]; then
	printf 'manifest kernelVersion is empty\n' >&2
	exit 1
fi
if [[ "${trust_embedded}" != "false" ]]; then
	printf 'manifest trustMaterial.distributionSpecificEmbedded = %q, want false\n' "${trust_embedded}" >&2
	exit 1
fi
if [[ "${manifest_name}" != "bootup-${release_version}-${arch}-manifest.json" ]]; then
	printf 'manifest name = %q, want bootup-%s-%s-manifest.json\n' \
		"${manifest_name}" "${release_version}" "${arch}" >&2
	exit 1
fi

sums_name="bootup-${release_version}-${arch}-SHA256SUMS"
sums_path="${release_dir}/${sums_name}"
if [[ ! -r "${sums_path}" ]]; then
	printf 'checksum file is not readable: %s\n' "${sums_path}" >&2
	exit 1
fi

declare -A roles_seen=()
iso_name=""

while IFS=$'\t' read -r role name bytes expected_sha256; do
	artifact_path="${release_dir}/${name}"
	if [[ -z "${role}" || -z "${name}" || -z "${bytes}" || -z "${expected_sha256}" ]]; then
		printf 'manifest contains an incomplete artifact entry\n' >&2
		exit 1
	fi
	if [[ "${name}" != *"${release_version}"* || "${name}" != *"${arch}"* ]]; then
		printf 'artifact name %q must include release %q and architecture %q\n' \
			"${name}" "${release_version}" "${arch}" >&2
		exit 1
	fi
	if [[ "${role}" == kernel-image || "${role}" == kernel-config ]]; then
		if [[ "${name}" != *"${kernel_version}"* ]]; then
			printf 'kernel artifact name %q must include kernel version %q\n' \
				"${name}" "${kernel_version}" >&2
			exit 1
		fi
	fi
	if [[ ! -r "${artifact_path}" ]]; then
		printf 'artifact is not readable: %s\n' "${artifact_path}" >&2
		exit 1
	fi
	actual_bytes="$(stat -c '%s' "${artifact_path}")"
	if [[ "${actual_bytes}" != "${bytes}" ]]; then
		printf 'artifact %s size = %s, manifest says %s\n' "${name}" "${actual_bytes}" "${bytes}" >&2
		exit 1
	fi
	actual_sha256="$(sha256_file "${artifact_path}")"
	if [[ "${actual_sha256}" != "${expected_sha256}" ]]; then
		printf 'artifact %s sha256 = %s, manifest says %s\n' \
			"${name}" "${actual_sha256}" "${expected_sha256}" >&2
		exit 1
	fi
	require_name_in_checksums "${sums_path}" "${name}"
	roles_seen["${role}"]=1
	if [[ "${role}" == "iso" ]]; then
		iso_name="${name}"
	fi
done < <(jq --raw-output '.artifacts[] | [.role, .name, (.bytes | tostring), .sha256] | @tsv' "${manifest_path}")

for role in bootup-binary kernel-image kernel-config initramfs iso; do
	if [[ -z "${roles_seen[${role}]:-}" ]]; then
		printf 'manifest is missing required role %s\n' "${role}" >&2
		exit 1
	fi
done

require_name_in_checksums "${sums_path}" "${manifest_name}"
(
	cd "${release_dir}"
	sha256sum --check "${sums_name}" >/dev/null
)

iso_path="${release_dir}/${iso_name}"
iso_listing="$(xorriso -indev "${iso_path}" -find / -type f -exec echo 2>/dev/null)"
require_iso_path "${iso_listing}" "/boot/bootup/vmlinuz"
require_iso_path "${iso_listing}" "/boot/bootup/initramfs.cpio.zst"
require_iso_path "${iso_listing}" "/boot/grub/grub.cfg"
require_iso_path "${iso_listing}" "/efi/boot/bootx64.efi"

printf 'release artifacts ok: %s\n' "${manifest_path}"
