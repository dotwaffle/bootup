#!/usr/bin/env bash
set -euo pipefail

usage() {
	cat >&2 <<'USAGE'
usage: scripts/smoke-iso-bios.sh [bootup.iso]

Environment:
  BOOTUP_ISO_SMOKE_TIMEOUT  timeout for QEMU, default 90s
  BOOTUP_ISO_SMOKE_EXPECT   regex that marks a successful bootup reach
USAGE
}

require_cmd() {
	if ! command -v "$1" >/dev/null 2>&1; then
		printf '%s not found; install %s to run the bootup ISO smoke\n' "$1" "$1" >&2
		exit 1
	fi
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
	usage
	exit 0
fi

require_cmd grep
require_cmd tail
require_cmd timeout

repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
iso="${1:-${BOOTUP_ISO:-dist/bootup.iso}}"
if [[ ! -r "${iso}" ]]; then
	printf 'ISO is not readable: %s\n' "${iso}" >&2
	exit 1
fi

timeout_value="${BOOTUP_ISO_SMOKE_TIMEOUT:-90s}"
expect="${BOOTUP_ISO_SMOKE_EXPECT:-bootup started.*mode=menu|bootup targets|Ubuntu 26[.]04 amd64 netboot|Debian trixie amd64 netboot}"
log="$(mktemp /tmp/bootup-iso-smoke.XXXXXX.log)"
cleanup() {
	rm -f "${log}"
}
trap cleanup EXIT

set +e
timeout --foreground "${timeout_value}" "${repo_root}/scripts/run-qemu-iso.sh" "${iso}" >"${log}" 2>&1
status=$?
set -e

if [[ "${status}" -ne 0 && "${status}" -ne 124 ]]; then
	printf 'QEMU ISO smoke failed with status %d\n' "${status}" >&2
	tail -n 80 "${log}" >&2
	exit "${status}"
fi

if ! grep -Eq "${expect}" "${log}"; then
	printf 'QEMU ISO smoke did not reach expected bootup output\n' >&2
	tail -n 120 "${log}" >&2
	exit 1
fi

printf 'iso BIOS smoke reached bootup: %s\n' "${iso}"
