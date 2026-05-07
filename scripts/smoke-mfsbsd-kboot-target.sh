#!/usr/bin/env bash
set -euo pipefail

usage() {
	cat >&2 <<'USAGE'
usage: scripts/smoke-mfsbsd-kboot-target.sh [--dry-run]

Build and run a UEFI QEMU smoke for the executable mfsBSD freebsd-kboot
catalog target. The smoke exercises the real bootup provider path: runtime
network prep, pinned mfsBSD ISO download, pinned FreeBSD base.txz download,
ISO extraction, loader.kboot staging, and the final mfsBSD memory-root boot.

Environment:
  BOOTUP_LIVE_MFSBSD_KBOOT_SMOKE=1
                         required opt-in for the live QEMU/network smoke
  BOOTUP_MFSBSD_KBOOT_TARGET
                         target ID, default mfsbsd-142-amd64
  BOOTUP_MFSBSD_KBOOT_KERNEL
                         bootup Linux kernel, default latest dist/kernel/*
  BOOTUP_MFSBSD_KBOOT_KERNEL_CONFIG
                         kernel config, default derived from kernel path
  BOOTUP_MFSBSD_KBOOT_WORKDIR
                         work directory, default /tmp/...
  BOOTUP_MFSBSD_KBOOT_TIMEOUT
                         QEMU timeout seconds, default 900
  BOOTUP_MFSBSD_KBOOT_LOG
                         serial log path, default workdir/qemu.log
  BOOTUP_MFSBSD_KBOOT_OVMF_CODE
                         OVMF CODE image, default /usr/share/OVMF/OVMF_CODE_4M.fd
  BOOTUP_MFSBSD_KBOOT_OVMF_VARS
                         OVMF VARS image, default CODE path with VARS
USAGE
}

require_cmd() {
	if ! command -v "$1" >/dev/null 2>&1; then
		printf '%s not found; install %s\n' "$1" "$1" >&2
		exit 2
	fi
}

latest_kernel() {
	local kernel_dir="$1"
	local -a kernels=()

	if [[ -d "${kernel_dir}" ]]; then
		mapfile -t kernels < <(find "${kernel_dir}" -maxdepth 1 -type f -name 'linux-*-bootup-amd64-bzImage' | sort -V)
	fi
	if [[ "${#kernels[@]}" -eq 0 ]]; then
		printf 'no bootup kernel found under %s; run scripts/build-kernel.sh or set BOOTUP_MFSBSD_KBOOT_KERNEL\n' "${kernel_dir}" >&2
		exit 2
	fi
	printf '%s\n' "${kernels[-1]}"
}

abs_path() {
	local input="$1"
	local dir
	dir="$(cd -- "$(dirname -- "${input}")" && pwd)"
	printf '%s/%s\n' "${dir}" "$(basename -- "${input}")"
}

require_log() {
	local pattern="$1"
	local description="$2"
	if grep -Eq "${pattern}" "${log}"; then
		return
	fi
	printf 'mfsBSD product smoke missing %s in %s\n' "${description}" "${log}" >&2
	exit 1
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
	usage
	exit 0
fi

dry_run=0
if [[ "${1:-}" == "--dry-run" ]]; then
	dry_run=1
	shift
fi
if [[ $# -ne 0 ]]; then
	usage
	exit 2
fi

if [[ "${BOOTUP_LIVE_MFSBSD_KBOOT_SMOKE:-}" != "1" ]]; then
	echo "BOOTUP_LIVE_MFSBSD_KBOOT_SMOKE=1 is required" >&2
	exit 2
fi

repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
target_id="${BOOTUP_MFSBSD_KBOOT_TARGET:-mfsbsd-142-amd64}"
if [[ ! "${target_id}" =~ ^[A-Za-z0-9_.-]+$ ]]; then
	printf 'target ID contains unsupported characters: %s\n' "${target_id}" >&2
	exit 2
fi

workdir="${BOOTUP_MFSBSD_KBOOT_WORKDIR:-}"
if [[ -z "${workdir}" && "${dry_run}" -eq 0 ]]; then
	workdir="$(mktemp -d /tmp/bootup-mfsbsd-kboot-target.XXXXXX)"
elif [[ -z "${workdir}" ]]; then
	workdir="/tmp/bootup-mfsbsd-kboot-target.dry-run"
fi
mkdir -p "${workdir}"
workdir="$(cd -- "${workdir}" && pwd)"

kernel="${BOOTUP_MFSBSD_KBOOT_KERNEL:-}"
if [[ -z "${kernel}" && "${dry_run}" -eq 0 ]]; then
	kernel="$(latest_kernel "${repo_root}/dist/kernel")"
elif [[ -z "${kernel}" ]]; then
	kernel="${repo_root}/dist/kernel/linux-bootup-amd64-bzImage"
fi
kernel_config="${BOOTUP_MFSBSD_KBOOT_KERNEL_CONFIG:-${kernel%-bzImage}.config}"

initramfs="${workdir}/bootup-mfsbsd-kboot-initramfs.cpio"
initramfs_zst="${initramfs}.zst"
iso="${workdir}/bootup-mfsbsd-kboot.iso"
log="${BOOTUP_MFSBSD_KBOOT_LOG:-${workdir}/qemu.log}"
timeout_s="${BOOTUP_MFSBSD_KBOOT_TIMEOUT:-900}"
ovmf_code="${BOOTUP_MFSBSD_KBOOT_OVMF_CODE:-/usr/share/OVMF/OVMF_CODE_4M.fd}"
ovmf_vars="${BOOTUP_MFSBSD_KBOOT_OVMF_VARS:-${ovmf_code/CODE/VARS}}"
cmdline="${BOOTUP_MFSBSD_KBOOT_CMDLINE:-console=tty0 console=ttyS0,115200n8 earlyprintk=ttyS0,115200 panic=30 ip=::::::dhcp}"
uinitcmd="bootup --mode=boot-target --target=${target_id} --staging-dir=/tmp/bootup --prepare-runtime"
target_pattern="${BOOTUP_MFSBSD_KBOOT_TARGET_PATTERN:-login:}"

if [[ "${dry_run}" -eq 1 ]]; then
	cat <<EOF
scripts/build-initramfs.sh ${initramfs} '${uinitcmd}' '${BOOTUP_MFSBSD_KBOOT_GO_TAGS:-}' '${BOOTUP_MFSBSD_KBOOT_EXTRA_FILES:-}'
scripts/build-iso.sh ${iso}
scripts/run-qemu-iso.sh ${iso}
target marker: ${target_pattern}
expected markers:
  FreeBSD/amd64 kboot loader
  UEFI SYSTAB PA
  UEFI MMAP
  Start @
  md0: Preloaded
  Trying to mount root from ufs:/dev/md0
  Dual Console: Serial Primary
  FreeBSD/amd64 (mfsbsd)
  login:
EOF
	exit 0
fi

require_cmd grep
require_cmd setsid
require_cmd qemu-system-x86_64

if [[ ! -r "${kernel}" ]]; then
	printf 'bootup kernel is not readable: %s\n' "${kernel}" >&2
	exit 2
fi
kernel="$(abs_path "${kernel}")"
if [[ ! -r "${kernel_config}" ]]; then
	printf 'bootup kernel config is not readable: %s\n' "${kernel_config}" >&2
	exit 2
fi
BOOTUP_KERNEL_CONFIG_REQUIRE_ISO_MOUNT=0 \
	"${repo_root}/scripts/check-kernel-config.sh" "${kernel_config}"
if [[ ! -r "${ovmf_code}" ]]; then
	printf 'OVMF CODE image is not readable: %s\n' "${ovmf_code}" >&2
	exit 2
fi
if [[ ! -r "${ovmf_vars}" ]]; then
	printf 'OVMF VARS image is not readable: %s\n' "${ovmf_vars}" >&2
	exit 2
fi

BOOTUP_INITRAMFS_ZSTD="${initramfs_zst}" \
	"${repo_root}/scripts/build-initramfs.sh" \
	"${initramfs}" \
	"${uinitcmd}" \
	"${BOOTUP_MFSBSD_KBOOT_GO_TAGS:-}" \
	"${BOOTUP_MFSBSD_KBOOT_EXTRA_FILES:-}"

BOOTUP_ISO_KERNEL="${kernel}" \
BOOTUP_ISO_INITRAMFS="${initramfs_zst}" \
BOOTUP_ISO_CMDLINE="${cmdline}" \
	"${repo_root}/scripts/build-iso.sh" "${iso}"

printf 'workdir: %s\n' "${workdir}"
printf 'serial log: %s\n' "${log}"

set +e
setsid env \
	BOOTUP_MEMORY="${BOOTUP_MEMORY:-4096}" \
	BOOTUP_QEMU_FIRMWARE="${ovmf_code}" \
	BOOTUP_QEMU_FIRMWARE_VARS="${ovmf_vars}" \
	"${repo_root}/scripts/run-qemu-iso.sh" "${iso}" >"${log}" 2>&1 &
qemu_group="$!"
set -e

deadline=$((SECONDS + timeout_s))
reached=0
while kill -0 "${qemu_group}" >/dev/null 2>&1; do
	if grep -Eq "${target_pattern}" "${log}" 2>/dev/null; then
		reached=1
		kill -TERM "-${qemu_group}" >/dev/null 2>&1 || true
		break
	fi
	if ((SECONDS >= deadline)); then
		kill -TERM "-${qemu_group}" >/dev/null 2>&1 || true
		break
	fi
	sleep 2
done

set +e
wait "${qemu_group}"
qemu_status="$?"
set -e

if grep -Eq "Can't find symbol boot_params|Can't get UEFI memory map" "${log}" 2>/dev/null; then
	printf 'mfsBSD product smoke hit the Linux boot metadata blocker.\n' >&2
	exit 20
fi

require_log "bootup started.*mode=boot-target" "bootup boot-target start"
require_log "\\[staging\\] mfsBSD 14\\.2 amd64" "mfsBSD staging status"
require_log "loader[[:space:]]+/tmp/bootup/.*/loader\\.kboot" "staged loader.kboot"
require_log "loader_archive[[:space:]]+/tmp/bootup/base\\.txz" "staged FreeBSD loader archive"
require_log "payload[[:space:]]+/tmp/bootup/mfsbsd-14\\.2-RELEASE-amd64\\.iso" "staged mfsBSD ISO"
require_log "payload_root[[:space:]]+/tmp/bootup/mfsbsd-root-" "staged mfsBSD payload root"
require_log "FreeBSD/amd64 kboot loader" "loader.kboot banner"
require_log "UEFI SYSTAB PA" "UEFI system table metadata"
require_log "UEFI MMAP" "UEFI memory map metadata"
require_log "Start @" "FreeBSD kernel jump"
require_log "md0: Preloaded image </mfsroot>" "preloaded mfsroot"
require_log "Trying to mount root from ufs:/dev/md0" "mfsBSD md root mount"
require_log "Dual Console: Serial Primary" "serial console handoff"
require_log "FreeBSD/amd64 \\(mfsbsd\\)" "mfsBSD login banner"
require_log "login:" "mfsBSD login prompt"

if [[ "${reached}" -ne 1 ]]; then
	printf 'mfsBSD product smoke did not reach target marker %s\n' "${target_pattern}" >&2
	exit 1
fi
if [[ "${qemu_status}" -ne 0 && "${qemu_status}" -ne 143 ]]; then
	printf 'mfsBSD product smoke QEMU exited with status %s after target marker\n' "${qemu_status}" >&2
	exit "${qemu_status}"
fi

printf 'mfsBSD product kboot smoke reached target marker.\n'
