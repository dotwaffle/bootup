#!/usr/bin/env bash
set -euo pipefail

usage() {
	cat >&2 <<'USAGE'
usage: scripts/run-qemu-iso.sh [bootup.iso]

Environment:
  BOOTUP_ISO            ISO path when no argument is supplied
  BOOTUP_MEMORY         QEMU memory in MiB, default 2048
  BOOTUP_QEMU_FIRMWARE  optional firmware image, for example OVMF_CODE_4M.fd
  BOOTUP_QEMU_FIRMWARE_VARS
                         optional OVMF vars image when using pflash firmware
USAGE
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
	usage
	exit 0
fi

iso="${1:-${BOOTUP_ISO:-dist/bootup.iso}}"
if [[ ! -r "${iso}" ]]; then
	printf 'ISO is not readable: %s\n' "${iso}" >&2
	exit 1
fi

qemu_args=(
	-m "${BOOTUP_MEMORY:-2048}"
	-nographic
	-boot d
	-cdrom "${iso}"
)
tmp_vars=""
cleanup() {
	if [[ -n "${tmp_vars}" ]]; then
		rm -f "${tmp_vars}"
	fi
}
trap cleanup EXIT

if [[ -n "${BOOTUP_QEMU_FIRMWARE:-}" ]]; then
	if [[ ! -r "${BOOTUP_QEMU_FIRMWARE}" ]]; then
		printf 'firmware is not readable: %s\n' "${BOOTUP_QEMU_FIRMWARE}" >&2
		exit 1
	fi
	if [[ "$(basename -- "${BOOTUP_QEMU_FIRMWARE}")" == *CODE* ]]; then
		vars="${BOOTUP_QEMU_FIRMWARE_VARS:-${BOOTUP_QEMU_FIRMWARE/CODE/VARS}}"
		if [[ ! -r "${vars}" ]]; then
			printf 'firmware vars are not readable: %s\n' "${vars}" >&2
			exit 1
		fi
		tmp_vars="$(mktemp /tmp/bootup-ovmf-vars.XXXXXX.fd)"
		cp "${vars}" "${tmp_vars}"
		qemu_args=(
			-drive "if=pflash,format=raw,unit=0,readonly=on,file=${BOOTUP_QEMU_FIRMWARE}"
			-drive "if=pflash,format=raw,unit=1,file=${tmp_vars}"
			"${qemu_args[@]}"
		)
	else
		qemu_args=(-bios "${BOOTUP_QEMU_FIRMWARE}" "${qemu_args[@]}")
	fi
fi

qemu-system-x86_64 "${qemu_args[@]}"
