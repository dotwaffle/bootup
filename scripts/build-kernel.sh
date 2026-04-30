#!/usr/bin/env bash
set -euo pipefail

usage() {
	echo "usage: $0 [output-dir]" >&2
}

latest_stable_kernel() {
	if ! command -v curl >/dev/null 2>&1; then
		printf 'curl not found; install curl or set BOOTUP_KERNEL_VERSION\n' >&2
		exit 1
	fi
	if ! command -v jq >/dev/null 2>&1; then
		printf 'jq not found; install jq or set BOOTUP_KERNEL_VERSION\n' >&2
		exit 1
	fi

	curl --fail --silent --show-error --location https://www.kernel.org/releases.json \
		| jq --exit-status --raw-output '.latest_stable.version'
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
	usage
	exit 0
fi

repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
out_dir="${1:-${BOOTUP_KERNEL_OUT:-dist/kernel}}"
kernel_version="${BOOTUP_KERNEL_VERSION:-$(latest_stable_kernel)}"
if [[ ! "${kernel_version}" =~ ^[0-9]+\.[0-9]+(\.[0-9]+)?$ ]]; then
	echo "bad kernel version: ${kernel_version}" >&2
	exit 1
fi

if ! command -v docker >/dev/null 2>&1; then
	echo "docker not found; install Docker to build the bootup kernel" >&2
	exit 1
fi

mkdir -p "${out_dir}"
tmp="$(mktemp -d "${out_dir}/kernel.XXXXXX")"
cleanup() {
	rm -rf "${tmp}"
}
trap cleanup EXIT

DOCKER_BUILDKIT=1 docker build \
	--build-arg "LINUX_VERSION=${kernel_version}" \
	--output "type=local,dest=${tmp}" \
	-f "${repo_root}/test/vmtest/kernel-amd64/Dockerfile" \
	"${repo_root}"

kernel_out="${out_dir}/linux-${kernel_version}-bootup-amd64-bzImage"
config_out="${out_dir}/linux-${kernel_version}-bootup-amd64.config"
install -m 0644 "${tmp}/bzImage" "${kernel_out}"
install -m 0644 "${tmp}/config_linux.resolved" "${config_out}"

stat -c "kernel %n %s bytes" "${kernel_out}"
stat -c "config %n %s bytes" "${config_out}"
trap - EXIT
cleanup
